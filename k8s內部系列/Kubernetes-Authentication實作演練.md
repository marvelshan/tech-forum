## K8S Lab Day_49

# Study9-Kubernetes-Authentication實作演練

## 前言

這幾天看了那麼多 source code，都沒有去實際去操作他，好像都不太確定對不對，今天就來試試看去 clone kubernetes 來去實際呼叫他，直接去使用它，這樣了解比較不會像是只單純看 code，還是要實際運作才會去了解他～

## Authentication

今天要來實際去了解在建立一個資源的時候會經過 authentication 來去驗證使用者，也就是 [`k8s.io/apiserver/pkg/authentication` 這個資料夾](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/apiserver/pkg/authentication)的行為，那我們就做個小實驗來看看實際的運作吧

```bash
cd kubernetes
mkdir -p _test/auth
touch _test/auth/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"k8s.io/klog/v2"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/pkg/authentication/request/union"
	"k8s.io/apiserver/pkg/authentication/user"
)

func main() {
	klog.InitFlags(nil)
    defer klog.Flush()
	// === 1. 建立一個假的 Token Authenticator ===
	// 模擬：任何 token 為 "valid-token" 都認證為 user "alice"
	fakeTokenAuth := authenticator.TokenFunc(func(ctx context.Context, token string) (*authenticator.Response, bool, error) {
		if token == "valid-token" {
			return &authenticator.Response{
				User: &user.DefaultInfo{
					Name:   "alice",
					UID:    "user-123",
					Groups: []string{"developers", "system:authenticated"},
					Extra:  map[string][]string{"team": {"backend"}},
				},
			}, true, nil
		}
		return nil, false, nil
	})

	// === 2. 包裝成 Bearer Token Authenticator ===
	bearerAuth := bearertoken.New(fakeTokenAuth)

	// === 3. 建立 Union Authenticator（可加入更多）===
	unionAuth := union.New(bearerAuth)

	// === 4. 建立假 HTTP Request ===
	req, _ := http.NewRequest("GET", "/api/v1/pods", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	// === 5. 執行 AuthenticateRequest ===
	resp, ok, err := unionAuth.AuthenticateRequest(req)

	if err != nil {
		log.Fatalf("Authentication error: %v", err)
	}
	if !ok {
		log.Println("Authentication failed: no authenticator accepted")
		os.Exit(1)
	}

	// === 6. 輸出結果 ===
	fmt.Println("Authentication SUCCESS!")
	fmt.Printf("User: %s\n", resp.User.GetName())
	fmt.Printf("UID:  %s\n", resp.User.GetUID())
	fmt.Printf("Groups: %v\n", resp.User.GetGroups())
	fmt.Printf("Extra:  %v\n", resp.User.GetExtra())
}
```

這邊可以試者把 `resp` print 出來，就可以得到 `&{[] 0x1400011c880}` 的 pointer，為何會得到這個呢？我們就要[往回看](https://github.com/kubernetes/kubernetes/blob/e5227216c0796d725c695e36cfc1d54e7631d3a6/staging/src/k8s.io/apiserver/pkg/authentication/request/union/union.go)到 `union.New(bearerAuth)`

```go
// New returns a request authenticator that validates credentials using a chain of authenticator.Request objects.
// The entire chain is tried until one succeeds. If all fail, an aggregate error is returned.
func New(authRequestHandlers ...authenticator.Request) authenticator.Request {
	if len(authRequestHandlers) == 1 {
		return authRequestHandlers[0]
	}
	return &unionAuthRequestHandler{Handlers: authRequestHandlers, FailOnError: false}
}
```

這裏的 return type 是 `authenticator.Request` 這邊就可以看到 [interface 的定義](https://github.com/kubernetes/kubernetes/blob/df292749c9d063b06861d0f4f1741c37b815a2fa/staging/src/k8s.io/apiserver/pkg/authentication/authenticator/interfaces.go)

```go
// Request attempts to extract authentication information from a request and
// returns a Response or an error if the request could not be checked.
type Request interface {
	AuthenticateRequest(req *http.Request) (*Response, bool, error)
}
// ...
type Response struct {
	// Audiences is the set of audiences the authenticator was able to validate
	// the token against. If the authenticator is not audience aware, this field
	// will be empty.
	Audiences Audiences
	// User is the UserInfo associated with the authentication context.
	User user.Info
}
```

這裏從 http.Request 獲取我們所需要的資訊，而這裏成功時就會回傳 ` *Response` 這裡就可以看到 `user.Info` 裡面就是我們所要[輸出的資料](https://github.com/kubernetes/kubernetes/blob/2ad2bd8907d979f709cd924af7986be71c31ce12/staging/src/k8s.io/apiserver/pkg/authentication/user/user.go)

```go
type Info interface {
	// GetName returns the name that uniquely identifies this user among all
	// other active users.
	GetName() string
	// GetUID returns a unique value for a particular user that will change
	// if the user is removed from the system and another user is added with
	// the same name.
	GetUID() string
	// GetGroups returns the names of the groups the user is a member of
	GetGroups() []string

	// GetExtra can contain any additional information that the authenticator
	// thought was interesting.  One example would be scopes on a token.
	// Keys in this map should be namespaced to the authenticator or
	// authenticator/authorizer pair making use of them.
	// For instance: "example.org/foo" instead of "foo"
	// This is a map[string][]string because it needs to be serializeable into
	// a SubjectAccessReviewSpec.authorization.k8s.io for proper authorization
	// delegation flows
	// In order to faithfully round-trip through an impersonation flow, these keys
	// MUST be lowercase.
	GetExtra() map[string][]string
}
```

## 結語

今天算是第一次把想要的東西印出來，前幾天真的算是紙上談兵，今天才真的有在實作的感覺，這樣比較有實際知道在幹嘛，而且也花了很久的時間在搞怎麼把 go.mod 裡面的 k8s.io/api => ./staging/src/k8s.io/api，跑了蠻多的方法，今天已經把環境 setting 好，明天就好好的幫 k8s 開腸剖腹！
