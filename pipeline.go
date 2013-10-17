package main

type Pipeline []Stage

func CreatePipeline() *Pipeline {
	pipe := make(Pipeline, 0)
	return &pipe
}

func (pipe *Pipeline) AddStage(stage Stage) {
	*pipe = append(*pipe, stage)
}

func (pipe *Pipeline) Execute(verbose bool) Diag {
	for _, stage := range *pipe {
		diag := stage.Run()
		if diag != nil {
			return diag
		}
	}
	return nil
}

type Stage interface {
	Name() string
	Run() Diag
}
