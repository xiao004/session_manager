package memory

import (
	"container/list"
	"session_manager/session"
	"sync"
	"time"
)


var pder = &Provider{list: list.New()}


// 实现了Session接口
type SessionStore struct {
	// session id, 唯一标识
	sid string
	// 最后访问时间
	timeAccessed time.Time
	// session里面存储的值
	// key(session名称) => value(session值) 映射
	value map[interface{}]interface{}
}


// set session value
func (st *SessionStore) Set(key, value interface{}) error {
	st.value[key] = value
	// 更新最近访问时间
	pder.SessionUpdate(st.sid)
	return nil
}


// get session value
func (st *SessionStore) Get(key interface{}) interface{} {
	// 更新最近访问时间
	pder.SessionUpdate(st.sid)
	if v, ok := st.value[key]; ok {
		return v
	} else {
		return nil
	}
}


// 删除key对应的session
func (st *SessionStore) Delete(key interface{}) error {
	// 从value map中删除key对应的对象
	delete(st.value, key)
	// 更新最近访问时间
	pder.SessionUpdate(st.sid)
	return nil
}


// 获取SessionStore对象中的sid
func (st *SessionStore) SessionID() string {
	return st.sid
}


// 底层存储结构
// 实现了Provider接口
type Provider struct {
	lock sync.Mutex
	// sid到list中元素的映射，list.Element为list的元素结构体类型
	sessions map[string] *list.Element
	// 用来做gc，list中的元素为SessionStore对象
	list *list.List
}


// 实现Session的初始化，操作成功则返回此新的Session接口类型变量
func (pder *Provider) SessionInit(sid string) (session.Session, error) {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	v := make(map[interface{}]interface{}, 0)
	newsess := &SessionStore{sid: sid, timeAccessed: time.Now(), value: v}
	element := pder.list.PushBack(newsess)
	pder.sessions[sid] = element
	return newsess, nil
}


// SessionRead函数返回sid所代表的Session接口类型变量
// 如果不存在，那么将以sid为参数调用SessionInit函数创建并返回一个新的Session接口类型变量
func (pder *Provider) SessionRead(sid string) (session.Session, error) {
	if element, ok := pder.sessions[sid]; ok {
		return element.Value.(*SessionStore), nil
	} else {
		// 不存储在，用sid初始化一个
		sess, err := pder.SessionInit(sid)
		return sess, err
	}
	return nil, nil
}


// SessionDestroy函数用来销毁sid对应的Session变量
func (pder *Provider) SessionDestroy(sid string) error {
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
		return nil
	}
	return nil
}


// SessionGC根据maxLifeTime来删除过期的数据
func (pder *Provider) SessionGC(maxlifetime int64) {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	for {
		// Provider.list中的元素是按照最近访问时间倒序排的
		// 因此我们只需要从后往前遍历list即释放超时的SessionStore即可
		element := pder.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*SessionStore).timeAccessed.Unix() + maxlifetime < time.Now().Unix()) {
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*SessionStore).sid)

		} else {
			// 从后往前遇到第一个没有过期的SesstionStore
			// 则前面的SesstionStore也一定没有过期
			// 这里break即可
			break;
		}
	}
}


// 更新sid最新访问时间，并维护list中的元素顺序（上一次访问时间越靠近当前则越靠近list首部
func (pder *Provider) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*SessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
		return nil
	}
	return nil
}


func init() {
	pder.sessions = make(map[string] *list.Element, 0)
	session.Register("memory", pder)
}



