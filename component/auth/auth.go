package auth

import (
	"fmt"
	"github.com/Dreamacro/clash/params"
	"sync"
)

type Authenticator interface {
	Verify(ctx *params.ValueContext, user string, pass string) bool
	VerifyByIp(ctx *params.ValueContext) (authed bool, isSupport bool)
	Users() []string
}

type AuthUser struct {
	User string
	Pass string
}

type inMemoryAuthenticator struct {
	storage   *sync.Map
	usernames []string
}

func (au *inMemoryAuthenticator) Verify(ctx *params.ValueContext, user string, pass string) bool {
	ctx.WithValue(params.ProxyUser, fmt.Sprintf("%s:%s", user, pass))
	realPass, ok := au.storage.Load(user)
	return ok && realPass == pass
}

func (au *inMemoryAuthenticator) VerifyByIp(ctx *params.ValueContext) (bool, bool) {
	return true, true
}

func (au *inMemoryAuthenticator) Users() []string { return au.usernames }

func NewAuthenticator(users []AuthUser) Authenticator {
	if len(users) == 0 {
		return nil
	}

	au := &inMemoryAuthenticator{storage: &sync.Map{}}
	for _, user := range users {
		au.storage.Store(user.User, user.Pass)
	}
	usernames := make([]string, 0, len(users))
	au.storage.Range(func(key, value any) bool {
		usernames = append(usernames, key.(string))
		return true
	})
	au.usernames = usernames

	return au
}
