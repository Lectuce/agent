package tool

type ToolHandler func(map[string]any) (string, error)

var ToolHandlers = map[string]ToolHandler{
	"bash":       runBash,
	"read_file":  runRead,
	"write_file": runWrite,
	"edit_file":  runEdit,
	"glob":       runGlob,
	"todo_write": runTodoWrite,
	"load_skill": loadSkill,
	"calculator": runCalculator,
}

var SubHandlers = map[string]ToolHandler{
	"bash":       runBash,
	"read_file":  runRead,
	"write_file": runWrite,
	"edit_file":  runEdit,
	"glob":       runGlob,
	"calculator": runCalculator,
}
