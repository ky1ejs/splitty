package main

import (
	"fmt"

	"github.com/kylejs/splitty/backend/internal/config"
)

func main() {
	cfg := config.Load()
	fmt.Printf("splitty server starting in %s mode\n", cfg.Env)
}
