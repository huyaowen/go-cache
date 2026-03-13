package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
)

// User 用户模型
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// userServiceImpl 用户服务实现
type userServiceImpl struct {
	mu     sync.RWMutex
	users  map[int]*User
	nextID int
}

func newUserService() *userServiceImpl {
	svc := &userServiceImpl{
		users:  make(map[int]*User),
		nextID: 1,
	}
	svc.users[1] = &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	svc.users[2] = &User{ID: 2, Name: "Bob", Email: "bob@example.com"}
	svc.users[3] = &User{ID: 3, Name: "Charlie", Email: "charlie@example.com"}
	svc.nextID = 4
	return svc
}

// GetUser 获取用户 - 带缓存注解
// @cacheable(cache="users", key="#0", ttl="300")
func (s *userServiceImpl) GetUser(id int) *User {
	log.Printf("🔍 [DB] Fetching user id=%d", id)
	time.Sleep(100 * time.Millisecond)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id]
}

// CreateUser 创建用户 - 带缓存更新注解
// @cacheput(cache="users", key="#result.ID", ttl="300")
func (s *userServiceImpl) CreateUser(name, email string) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := &User{ID: s.nextID, Name: name, Email: email}
	s.users[user.ID] = user
	s.nextID++
	log.Printf("💾 [DB] Created user id=%d", user.ID)
	return user
}

// DeleteUser 删除用户 - 带缓存失效注解
// @cacheevict(cache="users", key="#id")
func (s *userServiceImpl) DeleteUser(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.users[id]
	if exists {
		delete(s.users, id)
		log.Printf("🗑️ [DB] Deleted user id=%d", id)
	}
	return exists
}

// 全局变量（直接调用模式）
var (
	userService  *userServiceImpl   // 原始服务
	proxyService proxy.Proxy        // 代理对象
	usersCache   backend.CacheBackend
)

// GetUser 通过代理获取用户
func GetUser(id int) *User {
	log.Printf("DEBUG: GetUser called, proxyService=%v", proxyService != nil)
	if proxyService != nil {
		results := proxyService.Call("GetUser", []reflect.Value{reflect.ValueOf(id)})
		log.Printf("DEBUG: Call returned %d results", len(results))
		if len(results) > 0 {
			if user, ok := results[0].Interface().(*User); ok {
				return user
			}
		}
	}
	return userService.GetUser(id)
}

// CreateUser 通过代理创建用户
func CreateUser(name, email string) *User {
	if proxyService != nil {
		results := proxyService.Call("CreateUser", []reflect.Value{reflect.ValueOf(name), reflect.ValueOf(email)})
		if len(results) > 0 {
			if user, ok := results[0].Interface().(*User); ok {
				return user
			}
		}
	}
	return userService.CreateUser(name, email)
}

// DeleteUser 通过代理删除用户
func DeleteUser(id int) bool {
	if proxyService != nil {
		results := proxyService.Call("DeleteUser", []reflect.Value{reflect.ValueOf(id)})
		if len(results) > 0 {
			if deleted, ok := results[0].Interface().(bool); ok {
				return deleted
			}
		}
	}
	return userService.DeleteUser(id)
}

func init() {
	// 创建缓存管理器
	manager := core.NewCacheManager()

	// 创建原始服务
	userService = newUserService()

	// 创建代理
	factory := proxy.NewProxyFactory(manager)
	proxyObj, err := factory.Create(userService)
	if err != nil {
		log.Printf("Warning: Could not create proxy: %v", err)
		proxyService = nil
	} else {
		proxyService = proxyObj.(proxy.Proxy)
		
		// 注册注解到拦截器
		annotations := proxy.GetRegisteredAnnotations("userServiceImpl")
		if annotations != nil {
			for methodName, annotation := range annotations {
				proxyService.RegisterAnnotation(methodName, annotation)
			}
		}
	}

	// 获取缓存引用
	usersCache, _ = manager.GetCache("users")

	log.Println("✅ UserService initialized with @cacheable annotations")
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   Go-Cache Framework Web Demo          ║")
	fmt.Println("║   Annotation-based (@cacheable)        ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🚀 Server: http://localhost:8083")
	fmt.Println()
	fmt.Println("📡 APIs:")
	fmt.Println("   GET  /api/user/:id       - Get user (@cacheable)")
	fmt.Println("   POST /api/user           - Create user (@cacheput)")
	fmt.Println("   DELETE /api/user/:id     - Delete user (@cacheevict)")
	fmt.Println("   GET  /api/stats          - Cache stats")
	fmt.Println("   GET  /api/benchmark/:id  - Benchmark")
	fmt.Println()

	http.HandleFunc("/api/user/", handleGetUser)
	http.HandleFunc("/api/user", handleUser)
	http.HandleFunc("/api/stats", handleStats)
	http.HandleFunc("/api/benchmark/", handleBenchmark)
	http.HandleFunc("/", handleHome)

	log.Fatal(http.ListenAndServe(":8083", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html><head><title>Go-Cache Demo</title>
<style>
body{font-family:Arial,sans-serif;max-width:800px;margin:50px auto;padding:20px;background:#f5f5f5}
h1{color:#333}.card{background:white;padding:25px;margin:20px 0;border-radius:10px;box-shadow:0 2px 10px rgba(0,0,0,0.1)}
.endpoint{background:#f8f9fa;border-left:4px solid #007bff;padding:15px;margin:15px 0;border-radius:5px}
button{background:#007bff;color:white;border:none;padding:10px 20px;cursor:pointer;border-radius:5px;margin:5px}
button:hover{background:#0056b3}button.danger{background:#dc3545}button.danger:hover{background:#c82333}
#result{background:#1e1e1e;color:#d4d4d4;padding:20px;border-radius:8px;display:none;margin-top:20px}
pre{margin:0;white-space:pre-wrap;font-family:'Consolas',monospace}
.log{font-size:12px;color:#7f8c8d}
</style></head>
<body>
<h1>🚀 Go-Cache Framework Demo</h1>
<p>Annotation-based caching with @cacheable</p>
<div class="card">
<h3>📡 API Endpoints</h3>
<div class="endpoint">
<strong>GET /api/user/:id</strong> <span class="log">(@cacheable)</span><br>
<button onclick="test('/api/user/1')">Get User 1</button>
<button onclick="test('/api/user/2')">Get User 2</button>
<button onclick="test('/api/user/3')">Get User 3</button>
</div>
<div class="endpoint">
<strong>GET /api/stats</strong> <span class="log">(Cache statistics)</span><br>
<button onclick="test('/api/stats')">View Stats</button>
</div>
<div class="endpoint">
<strong>GET /api/benchmark/:id</strong> <span class="log">(10 requests)</span><br>
<button onclick="test('/api/benchmark/1')">Run Benchmark</button>
</div>
</div>
<div id="result"><pre id="content"></pre></div>
<script>
async function test(url){
const start=performance.now();
const r=await fetch(url);const d=await r.json();
d._response_time_ms=(performance.now()-start).toFixed(2);
document.getElementById('content').textContent=JSON.stringify(d,null,2);
document.getElementById('result').style.display='block';
}
</script>
</body></html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "GET" {
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Method not allowed"})
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/user/")
	var id int
	fmt.Sscanf(idStr, "%d", &id)
	if id == 0 {
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Invalid user ID"})
		return
	}
	user := GetUser(id)
	if user == nil {
		json.NewEncoder(w).Encode(Response{Success: false, Error: "User not found"})
		return
	}
	json.NewEncoder(w).Encode(Response{Success: true, Data: user})
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		user := CreateUser(req.Name, req.Email)
		json.NewEncoder(w).Encode(Response{Success: true, Data: user})
	case "DELETE":
		idStr := strings.TrimPrefix(r.URL.Path, "/api/user/")
		var id int
		fmt.Sscanf(idStr, "%d", &id)
		deleted := DeleteUser(id)
		json.NewEncoder(w).Encode(Response{Success: deleted, Data: map[string]bool{"deleted": deleted}})
	default:
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Method not allowed"})
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := usersCache.Stats()
	json.NewEncoder(w).Encode(Response{Success: true, Data: map[string]interface{}{
		"cache":    "users",
		"hits":     stats.Hits,
		"misses":   stats.Misses,
		"sets":     stats.Sets,
		"hit_rate": fmt.Sprintf("%.2f%%", stats.HitRate*100),
		"size":     stats.Size,
	}})
}

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := strings.TrimPrefix(r.URL.Path, "/api/benchmark/")
	var id int
	fmt.Sscanf(idStr, "%d", &id)
	if id == 0 {
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Invalid user ID"})
		return
	}
	start := time.Now()
	for i := 0; i < 10; i++ {
		GetUser(id)
	}
	elapsed := time.Since(start)
	stats := usersCache.Stats()
	json.NewEncoder(w).Encode(Response{Success: true, Data: map[string]interface{}{
		"requests":   10,
		"total_time": elapsed.String(),
		"avg_time":   (elapsed / 10).String(),
		"hits":       stats.Hits,
		"misses":     stats.Misses,
		"hit_rate":   fmt.Sprintf("%.2f%%", stats.HitRate*100),
	}})
}


