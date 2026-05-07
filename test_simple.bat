@echo off
echo Testing go-lanlink room code generation...

go run -c "
package main
import (
    fmt
    github.com/yourusername/go-lanlink/pkg/protocol
)
func main() {
    code, err := protocol.GenerateRoomCode()
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Println("Generated room code:", code)

    normalized := protocol.NormalizeRoomCode(code)
    fmt.Println("Normalized:", normalized)

    valid := protocol.ValidateRoomCode(code)
    fmt.Println("Valid:", valid)
}
"

pause
