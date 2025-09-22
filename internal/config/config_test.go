package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWelcome(t *testing.T) {
	f, err := os.CreateTemp("", "atak_test")
	require.NoError(t, err)
	
	fmt.Fprint(f, "---\nwelcome_msg:\n    aaa\n")
	f.Close()
	
	c := NewAppConfig()
	c.Load(f.Name())
	
	require.Empty(t, c.WelcomeForScope("test"))
	require.Equal(t, "aaa", c.WelcomeMsg())
}

func TestWelcomeScope(t *testing.T) {
	f, err := os.CreateTemp("", "atak_test")
	require.NoError(t, err)
	
	fmt.Fprint(f, "---\nwelcome_msg:\n    test: \"aaa\"\n")
	f.Close()
	
	c := NewAppConfig()
	c.Load(f.Name())
	
	require.Empty(t, c.WelcomeMsg())
	require.Equal(t, "aaa", c.WelcomeForScope("test"))
}