package service

import (
	"context"
	"net/http"

	v1 "github.com/bizjs/kratoscarf/examples/api/auth/v1"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/biz"

	authjwt "github.com/bizjs/kratoscarf/auth/jwt"
	"github.com/bizjs/kratoscarf/auth/session"
	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// Compile-time check: AuthService implements the proto-generated interface.
var _ v1.AuthServiceHTTPServer = (*AuthService)(nil)

// AuthService implements v1.AuthServiceHTTPServer.
type AuthService struct {
	userUC  *biz.UserUsecase
	jwtAuth *authjwt.Authenticator
	sessMgr *session.Manager
}

func NewAuthService(
	userUC *biz.UserUsecase,
	jwtAuth *authjwt.Authenticator,
	sessMgr *session.Manager,
) *AuthService {
	return &AuthService{userUC: userUC, jwtAuth: jwtAuth, sessMgr: sessMgr}
}

func (s *AuthService) JWTAuth() *authjwt.Authenticator { return s.jwtAuth }
func (s *AuthService) SessionMgr() *session.Manager    { return s.sessMgr }

// ---------------------------------------------------------------------------
// JWT
// ---------------------------------------------------------------------------

func (s *AuthService) JWTLogin(ctx context.Context, req *v1.LoginRequest) (*v1.TokenPair, error) {
	user, err := s.userUC.Authenticate(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	pair, err := s.jwtAuth.GenerateTokenPair(ctx, authjwt.Claims{
		UserID:   user.ID,
		Username: user.Username,
	})
	if err != nil {
		return nil, err
	}
	return &v1.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *AuthService) JWTRefresh(ctx context.Context, req *v1.RefreshRequest) (*v1.TokenPair, error) {
	pair, err := s.jwtAuth.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &v1.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *AuthService) JWTProfile(ctx context.Context, _ *v1.EmptyRequest) (*v1.ProfileReply, error) {
	// ctx already carries Claims — the proto-generated handler calls
	// ctx.Middleware() which executes the JWT middleware chain.
	claims := authjwt.ClaimsFromContext(ctx)
	if claims == nil {
		return nil, biz.ErrUserNotFound
	}
	user, err := s.userUC.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.ProfileReply{Id: user.ID, Username: user.Username, Source: "jwt"}, nil
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

func (s *AuthService) SessionLogin(ctx context.Context, req *v1.LoginRequest) (*v1.ProfileReply, error) {
	user, err := s.userUC.Authenticate(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	httpReq, w := httpFromContext(ctx)
	if httpReq == nil {
		return nil, biz.ErrUserNotFound
	}

	sess, err := s.sessMgr.GetSession(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	sess.Set("userID", user.ID)
	sess.Set("username", user.Username)
	if err := s.sessMgr.SaveSession(ctx, w, sess); err != nil {
		return nil, err
	}
	return &v1.ProfileReply{Id: user.ID, Username: user.Username, Source: "session"}, nil
}

func (s *AuthService) SessionLogout(ctx context.Context, _ *v1.EmptyRequest) (*v1.MessageReply, error) {
	httpReq, w := httpFromContext(ctx)
	if httpReq == nil {
		return &v1.MessageReply{Message: "not an HTTP request"}, nil
	}
	if err := s.sessMgr.DestroySession(ctx, w, httpReq); err != nil {
		return nil, err
	}
	return &v1.MessageReply{Message: "logged out"}, nil
}

func (s *AuthService) SessionProfile(ctx context.Context, _ *v1.EmptyRequest) (*v1.ProfileReply, error) {
	// ctx already carries Session — the proto-generated handler calls
	// ctx.Middleware() which executes the Session middleware chain.
	sess := session.FromContext(ctx)
	if sess == nil || sess.IsNew {
		return nil, biz.ErrUserNotFound
	}
	userID, ok := sess.Get("userID")
	if !ok {
		return nil, biz.ErrUserNotFound
	}
	username, _ := sess.Get("username")
	return &v1.ProfileReply{
		Id:       userID.(string),
		Username: username.(string),
		Source:   "session",
	}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// httpFromContext extracts *http.Request and http.ResponseWriter from the
// Kratos server context (via transport).
func httpFromContext(ctx context.Context) (*http.Request, http.ResponseWriter) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil, nil
	}
	ht, ok := tr.(*kratoshttp.Transport)
	if !ok {
		return nil, nil
	}
	return ht.Request(), ht.Response()
}
