set GOOS=linux
set GOARCH=amd64

go build -o ./build/game-worker src/main.go

set GOOS=linux
set GOARCH=arm
set GOARM=7

go build -o ./build/game-worker-arm src/main.go