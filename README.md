# session_manager
this is a go language session manager

该项目通过加锁的方式保证了所有操作都是线程安全的

example:
``` go
package main

// 引入包
import (
	"fmt"
	"net/http"
	"session_manager/session"
	_ "session_manager/memory"
)

// 创建session管理对象
var globalSessions *session.Manager
// 在init中初始化该session管理对象
func init() {
	globalSessions, _ = session.NewManager("memory", "gosessionid", 3600)
	// 3600s后自动销毁该session
	go globalSessions.GC()
}

// 测试session <key value> 增删改查
func example(w http.ResponseWriter, r *http.Request) {
	// 将session关联到客户端
	sess := globalSessions.SessionStart(w, r)

	// 增加or修改一对key value
	sess.Set("key", "value")

	// 查询key对应的value
	value := sess.Get("key")
	fmt.Println(value)

	// 删除指定的key value
	sess.Delete("key")

	// 主动销毁session管理器
	session.SessionDestroy(w, r)
}

func main() {
	http.HandleFunc("/example", example)

	err := http.ListenAndServe(":8898", nil)

	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
```
