@echo off
REM Generate Go code from proto definitions.
REM Run from the proto/ directory: cd proto && generate.bat
REM
REM Prerequisites:
REM   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
REM   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
REM   protoc must be on PATH

echo Generating Go code from manga/manga.proto...

protoc --go_out=. --go_opt=paths=source_relative ^
       --go-grpc_out=. --go-grpc_opt=paths=source_relative ^
       manga/manga.proto

if %ERRORLEVEL% EQU 0 (
    echo Done! Generated files:
    echo   manga/manga.pb.go
    echo   manga/manga_grpc.pb.go
) else (
    echo ERROR: protoc failed. Make sure protoc, protoc-gen-go, and protoc-gen-go-grpc are installed.
    echo   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    echo   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    exit /b 1
)
