package main

import "os/exec"
//import "io"
//import "io/ioutil"
import "strings"

type CmdStage struct {
	Cmd  string
	Args []string
}

func CreateCmdStage(cmd string, args []string) *CmdStage {
	return &CmdStage{cmd, args}
}

func (stage *CmdStage) Name() string {
	return stage.Cmd
}

type CmdErr struct {
	Stage  *CmdStage
	Output string
}

func (err *CmdErr) Blame() Blame {
	invocation := err.Stage.Cmd + " " + strings.Join(err.Stage.Args, " ")
	return Blame{BLAME_CMD, "", 0, 0, 0, 0, 0, 0, 0, err.Stage.Cmd, invocation, err.Output}
}

func (err *CmdErr) Msg() string {
	return "Command exited with non-zero status."
}

func (stage *CmdStage) Run() Diag {
	cmd := exec.Command(stage.Cmd, stage.Args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CmdErr{stage, string(out)}
	}
	return nil
}

func CreateLLCStage(input string, output string) *CmdStage {
	cmd := "llc"
	args := []string{"-filetype=obj", "-o=" + output, input}
	return CreateCmdStage(cmd, args)
}

func CreateLinkStage(inputs []string, output string) *CmdStage {
	cmd := "clang"
	args := append(inputs, "-o", output)
	return CreateCmdStage(cmd, args)
}

func CreateCleanStage(clean ...string) *CmdStage {
	cmd := "rm"
	args := clean
	return CreateCmdStage(cmd, args)
}
