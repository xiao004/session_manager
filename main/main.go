package main

import (
	"fmt"
	"log"
	"net/http"
	"session_manager/session"
	_ "session_manager/memory"
)

// 创建session管理对象
var globalSessions *session.Manager
// 在init中初始化该session管理对象
func init() {
	globalSessions, _ = session.NewManager("memory", "gosessionid", 3600)
	go globalSessions.GC()
}


//  服务端读取cookie
func read_cookie(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	fmt.Println(r.Form)

	// 读取cookie方式1
	for _, cookie := range r.Cookies() {
		fmt.Fprint(w, cookie.Name + ": ")
		fmt.Fprint(w, cookie.Value + "\n")
		
		fmt.Println(cookie.Name + ": " + cookie.Value)
	}

	// // 读取cookie方式2
	// cookie, _ := r.Cookie("username")
	// fmt.Fprint(w, cookie )
}


// 服务端设置cookie
func set_cookie(w http.ResponseWriter, r *http.Request) {

	// 设置cookie
	cookie := http.Cookie{Name: "username", Value: "xiao004"} 
	http.SetCookie(w, &cookie)
}


// 使用session管理器case
func count(w http.ResponseWriter, r *http.Request) {
	// 将session关联到客户端
	sess := globalSessions.SessionStart(w, r)
	ct := sess.Get("countnum")
	if ct == nil {
		sess.Set("countnum", 1)
	} else {
		sess.Set("countnum", (ct.(int) + 1))
	}

	fmt.Println(sess.Get("countnum"))
	fmt.Fprint(w, "countnum: %d", sess.Get("countnum"))
}



func main() {
	http.HandleFunc("/read", read_cookie)
	http.HandleFunc("/set", set_cookie)
	http.HandleFunc("/count", count)

	err := http.ListenAndServe(":8898", nil)

	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
