package domain

import "time"

type User struct {
	ID        string    `json:"id"`
	OpenID    string    `json:"openid,omitempty"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar,omitempty"`
	Birthday  string    `json:"birthday,omitempty"`
	WxID      string    `json:"wxid,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Couple struct {
	ID           string    `json:"id"`
	UserAID      string    `json:"userAId"`
	UserBID      string    `json:"userBId"`
	LoveDate     string    `json:"loveDate"`
	PairCode     string    `json:"pairCode,omitempty"`
	CodeExpireAt time.Time `json:"codeExpireAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Anniversary struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Days  int    `json:"days"`
	Date  string `json:"date"`
	Tone  string `json:"tone"`
}

type Stat struct {
	Icon  string `json:"icon"`
	Value int    `json:"value"`
	Label string `json:"label"`
}

type Dashboard struct {
	LoveDays      int           `json:"loveDays"`
	Since         string        `json:"since"`
	Anniversaries []Anniversary `json:"anniversaries"`
	Stats         []Stat        `json:"stats"`
	ActiveGoals   int           `json:"activeGoals"`
	GoalProgress  int           `json:"goalProgress"`
	PendingTasks  int           `json:"pendingTasks"`
}

type Moment struct {
	ID        string    `json:"id"`
	CoupleID  string    `json:"coupleId"`
	AuthorID  string    `json:"authorId"`
	Author    string    `json:"author"`
	Avatar    string    `json:"avatar"`
	TimeLabel string    `json:"time"`
	Tag       string    `json:"tag"`
	Content   string    `json:"content"`
	Image     string    `json:"image,omitempty"`
	Liked     bool      `json:"liked"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskStatus string

const (
	TaskTodo   TaskStatus = "todo"
	TaskReview TaskStatus = "review"
	TaskDone   TaskStatus = "done"
)

type Task struct {
	ID       string     `json:"id"`
	CoupleID string     `json:"coupleId"`
	Title    string     `json:"title"`
	Owner    string     `json:"owner"`
	Type     string     `json:"type"`
	Tag      string     `json:"tag,omitempty"`
	Due      string     `json:"due,omitempty"`
	Reward   string     `json:"reward,omitempty"`
	Status   TaskStatus `json:"status"`
}

type ScheduledTask struct {
	ID       string `json:"id"`
	CoupleID string `json:"coupleId"`
	Title    string `json:"title"`
	Cycle    string `json:"cycle"`
	Assignee string `json:"assignee"`
	Time     string `json:"time"`
	Next     string `json:"next"`
	Pending  bool   `json:"pending"`
}

type Dish struct {
	ID       string `json:"id"`
	CoupleID string `json:"coupleId"`
	Icon     string `json:"icon"`
	Name     string `json:"name"`
	Meal     string `json:"meal"`
	Enabled  bool   `json:"enabled"`
}

type Order struct {
	ID       string   `json:"id"`
	CoupleID string   `json:"coupleId"`
	Date     string   `json:"date"`
	Meal     string   `json:"meal"`
	Picker   string   `json:"picker"`
	Dishes   []string `json:"dishes"`
}

type Goal struct {
	ID           string `json:"id"`
	CoupleID     string `json:"coupleId"`
	Title        string `json:"title"`
	Period       string `json:"period"`
	TargetValue  int    `json:"targetValue"`
	CurrentValue int    `json:"currentValue"`
	StartDate    string `json:"startDate"`
	TargetDate   string `json:"targetDate"`
	Progress     int    `json:"progress"`
	TimeProgress int    `json:"timeProgress"`
	RemainDays   int    `json:"remainDays"`
	Status       string `json:"status"`
}

type MePayload struct {
	Me      User   `json:"me"`
	Partner User   `json:"partner"`
	Couple  Couple `json:"couple"`
}

type DashboardPayload struct {
	Users     map[string]UserLite `json:"users"`
	Dashboard Dashboard           `json:"dashboard"`
}

type SyncState struct {
	Paired    bool      `json:"paired"`
	CoupleID  string    `json:"coupleId,omitempty"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type UserLite struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar,omitempty"`
	AvatarText string `json:"avatarText"`
	Birthday   string `json:"birthday"`
	WxID       string `json:"wxid"`
}

type AdminOverview struct {
	TotalUsers       int `json:"totalUsers"`
	PairedCouples    int `json:"pairedCouples"`
	PendingPairCodes int `json:"pendingPairCodes"`
	TotalMoments     int `json:"totalMoments"`
	OpenTasks        int `json:"openTasks"`
	ActiveGoals      int `json:"activeGoals"`
	TotalOrders      int `json:"totalOrders"`
	EnabledDishes    int `json:"enabledDishes"`
	ScheduledTasks   int `json:"scheduledTasks"`
	NewUsers24h      int `json:"newUsers24h"`
	NewCouples7d     int `json:"newCouples7d"`
}

type AdminUserSummary struct {
	ID        string    `json:"id"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	WxID      string    `json:"wxid"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminCoupleSummary struct {
	ID           string    `json:"id"`
	UserAID      string    `json:"userAId"`
	UserBID      string    `json:"userBId"`
	LoveDate     string    `json:"loveDate"`
	PairCode     string    `json:"pairCode,omitempty"`
	CodeExpireAt time.Time `json:"codeExpireAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AdminSystemPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	CPUPercent      float64   `json:"cpuPercent"`
	MemoryPercent   float64   `json:"memoryPercent"`
	ProcessMemoryMB float64   `json:"processMemoryMB"`
	Goroutines      int       `json:"goroutines"`
	RequestTotal    uint64    `json:"requestTotal"`
	ErrorTotal      uint64    `json:"errorTotal"`
}

type AdminRuntime struct {
	AppName           string    `json:"appName"`
	Version           string    `json:"version"`
	RunMode           string    `json:"runMode"`
	StartAt           time.Time `json:"startAt"`
	UptimeSeconds     int64     `json:"uptimeSeconds"`
	Goroutines        int       `json:"goroutines"`
	RequestTotal      uint64    `json:"requestTotal"`
	ErrorTotal        uint64    `json:"errorTotal"`
	LastCPUPercent    float64   `json:"lastCpuPercent"`
	LastMemoryPercent float64   `json:"lastMemoryPercent"`
	ProcessMemoryMB   float64   `json:"processMemoryMB"`
}

type AdminErrorLog struct {
	Time      string `json:"time"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error,omitempty"`
	Raw       string `json:"raw"`
}

type AdminDashboard struct {
	Overview      AdminOverview        `json:"overview"`
	Runtime       AdminRuntime         `json:"runtime"`
	Metrics       []AdminSystemPoint   `json:"metrics"`
	RecentUsers   []AdminUserSummary   `json:"recentUsers"`
	RecentCouples []AdminCoupleSummary `json:"recentCouples"`
	ErrorLogs     []AdminErrorLog      `json:"errorLogs"`
}
