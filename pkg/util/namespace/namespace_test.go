package namespace

import (
	"fmt"
	"testing"
)

func TestConfig_Execute(t *testing.T) {
	config := &Config{
		Mount:  true,    // Execute into mount namespace
		Target: 2300908, // Enter into Target namespace
	}

	stdout, stderr, err := config.Execute("nvidia-smi")
	if err != nil {
		fmt.Println(stderr)
		panic(err)
	}

	fmt.Println(stdout)
}
