module github.com/shiwa/timecard-mini/timebeat

go 1.21

require (
	github.com/elastic/beats/v7 v7.17.10
	github.com/shiwa/timecard-mini/tc-sync v0.0.0
)

replace github.com/shiwa/timecard-mini/tc-sync => ../tc-sync
