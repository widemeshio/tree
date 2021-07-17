package tree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionsApply(t *testing.T) {
	logger := NewDevelopmentLogger()
	options := Options{}
	options.Apply(WithName("task-one"), WithLogger(logger))
	require.Equal(t, logger, options.Logger)
	require.Equal(t, "task-one", options.Name)
}

func TestNewOptions(t *testing.T) {
	options := NewOptions(WithName("task-one"))
	require.Equal(t, "task-one", options.Name)
}

func TestClearNonInheritablesOptions(t *testing.T) {
	options := NewOptions(WithName("task-one"))
	require.Equal(t, "task-one", options.Name)
	options.Apply(clearNonInheritables())
	require.Empty(t, options.Name, "name is not inheritable")
}

func TestWithOptions(t *testing.T) {
	options := NewOptions(WithName("task-one"))
	require.Equal(t, "task-one", options.Name)
	options.Apply(WithOptions(Options{
		Name: "with-task-one",
	}))
	require.Equal(t, "with-task-one", options.Name)
}
