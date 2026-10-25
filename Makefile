.PHONY: run build clean

# 桌面端开发模式（Vite 热更新 + Go 热重启）
run:
	bun run dev

# 构建当前平台：build/<os>-<arch>/ 下 exe/安装包
build:
	bun run build

# 跨平台构建（无 cgo，任何机器可出全平台）
build-all:
	bun run build -- -platform windows/amd64,linux/amd64,darwin/arm64

clean:
	rm -rf build/ dist/ .mygo/
