package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// 使用该包的用户需要实现Session（session对业务接口）和Provider（底层存储接口）接口

// 操作session的业务接口，由用户自己根据具体业务来实现
type Session interface {
	// set session value
	Set(key, value interface{}) error
	// get session value
	Get(key interface{}) interface{}
	// delete session value
	Delete(key interface{}) error
	// back current sessionId
	SessionID() string
}


// session是保存在服务器端的数据，它可以以任何的方式存储，比如存储在内存、数据库或者文件中。
// 因此我们抽象出 一个Provider接口，用以表征session管理器底层存储结构
type Provider interface {
	// 实现Session的初始化，操作成功则返回此新的Session接口类型变量
	SessionInit(sid string) (Session, error)
	// SessionRead函数返回sid所代表的Session变量，如果不存在，那么将以sid为参数调用SessionInit函数创建并返回一个 新的Session变量
	SessionRead(sid string) (Session, error)
	// SessionDestroy函数用来销毁sid对应的Session变量
	SessionDestroy(sid string) error
	// SessionGC根据maxLifeTime来删除过期的数据
	SessionGC(maxLifeTime int64)
}


// 注册名称到Provider对象的映射
// 标记一个名称是否注册过
var provides = make(map[string]Provider)

// 注册函数，将要注册的名称映射到Provider对象上
// 已经注册过或者Provider对象为nil则抛出panic
func Register(name string, provide Provider) {
	if provide == nil {
		panic("session: Register provide is nil")
	}
	if _, dup := provides[name]; dup {
		panic("session: Register called twice for provide " + name)
	}
	provides[name] = provide
}


// session 管理器
type Manager struct {
	// private cookiename
	cookieName string
	// protects session
	lock sync.Mutex
	// session底层存储接口
	provider Provider
	// expiration time
	maxlifetime int64
}


// 创建Manager类对象
func NewManager(providerName, cookieName string, maxlifetime int64) (*Manager, error) {
	provider, ok := provides[providerName]
	if !ok {
		return nil, fmt.Errorf("session: unknown provide %q (forgotten import?)", providerName)
	}
	return &Manager{provider: provider, cookieName: cookieName, maxlifetime: maxlifetime}, nil
}


// get Session
// 检测是否已经有某个Session与当前来访用户发生了关联，如果没有则创建之
// 发生关联即将sid(Session id)，manager.cookieName设置到cookie中
// 指定返回session类型的session变量
func (manager *Manager) SessionStart(w http.ResponseWriter, r *http.Request) (session Session) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	// 读取客户端发过来的cookie
	cookie, err := r.Cookie(manager.cookieName)
	// 读取失败或者manager.cookieNmae为""即cookie还未与session发生关联
	if err != nil || cookie.Value == "" {
		// 该cookie还未与session发生关联
		// 创建一个全局唯一的session id
		sid := manager.sessionId()
		// session初始化，返回一个Session接口类型对象
		session, _ = manager.provider.SessionInit(sid)

		// 将session id，manager.cookieNmae设置到cookie中
		// 设置MaxAge=0后session cookie不会保存到浏览器历史记录中
		cookie := http.Cookie{Name: manager.cookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: true, MaxAge: int(manager.maxlifetime)}
		http.SetCookie(w, &cookie)

	} else {
		// 获取已关联的session id
		sid, _ := url.QueryUnescape(cookie.Value)
		// 通过 session id 获取 Session 接口对象
		session, _ = manager.provider.SessionRead(sid)
	}
	// 前面声明了返回的变量名称，这里直接写return即可
	return
}


// session重置
// 当用户退出应用的时候，我们需要对该用户的session数据进行销毁操作
func (manager *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		return
	} else {
		manager.lock.Lock()
		defer manager.lock.Unlock()

		manager.provider.SessionDestroy(cookie.Value)
		expiration := time.Now()

		// 设置该cookie为过期
		cookie := http.Cookie{Name: manager.cookieName, Path: "/", HttpOnly: true, Expires: expiration, MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}


// sesion自动销毁
// 使用方法
// 只要我们在Main启动的时候启动:
// func init() {
// 	go globalSessions.GC()
// }
func (manager *Manager) GC() {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	manager.provider.SessionGC(manager.maxlifetime)

	// 利用了time包中的定时器功能，当超时maxLifeTime之后调用GC函数，这样就可以保证maxLifeTime时间内
	// 的session都是可用的，类似的方案也可以用于统计在线用户数之类的
	time.AfterFunc(time.Duration(manager.maxlifetime), func(){manager.GC()})
}


// 创建全局唯一的session id
func (manager *Manager) sessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}


