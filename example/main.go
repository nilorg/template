package main

import (
	"fmt"
	"os"

	"github.com/nilorg/sdk/signal"
	"github.com/nilorg/template"
)

func main() {
	funMap := template.FuncMap{}
	en, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funMap)
	if err != nil {
		fmt.Printf("new engine error: %s\n", err)
		return
	}
	en.Init()
	err = en.Watching()
	if err != nil {
		fmt.Printf("new engine error: %s\n", err)
		return
	}
	en.RenderError(os.Stdout, "404", nil)
	fmt.Println("=========================")
	en.RenderSingle(os.Stdout, "login", nil)
	fmt.Println("=========================")
	en.RenderPage(os.Stdout, "posts/list", nil)
	signal.AwaitExit()
	en.Close()
}
