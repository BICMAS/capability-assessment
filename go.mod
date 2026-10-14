module github.com/bicmas-academy/learn

go 1.27

require (
	github.com/bicmas-academy/learn/services/assessment-gap v0.0.0
	github.com/bicmas-academy/learn/services/assessment-plan v0.0.0
	github.com/bicmas-academy/learn/services/audit v0.0.0
	github.com/bicmas-academy/learn/services/capability-framework v0.0.0
	github.com/bicmas-academy/learn/services/development v0.0.0
	github.com/bicmas-academy/learn/services/gateway v0.0.0
	github.com/bicmas-academy/learn/services/notify v0.0.0
	github.com/bicmas-academy/learn/services/org-directory v0.0.0
)

require github.com/go-chi/chi/v5 v5.2.3 // indirect

replace (
	github.com/bicmas-academy/learn/services/assessment-gap => ./services/assessment-gap
	github.com/bicmas-academy/learn/services/assessment-plan => ./services/assessment-plan
	github.com/bicmas-academy/learn/services/audit => ./services/audit
	github.com/bicmas-academy/learn/services/capability-framework => ./services/capability-framework
	github.com/bicmas-academy/learn/services/development => ./services/development
	github.com/bicmas-academy/learn/services/gateway => ./services/gateway
	github.com/bicmas-academy/learn/services/notify => ./services/notify
	github.com/bicmas-academy/learn/services/org-directory => ./services/org-directory
)
