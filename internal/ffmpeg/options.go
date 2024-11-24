package ffmpeg

import (
	"context"
	"os/exec"
	"strings"
)

type ffmpegBuilder struct {
	binary  string            // ffmpeg binary path
	options map[string]string // global options
	inputs  []*inputBuilder   // input options
	outputs []*outputBuilder  // output options
}

func newFFmpegBuilder() *ffmpegBuilder {
	return &ffmpegBuilder{
		binary:  "ffmpeg",
		options: make(map[string]string),
	}
}

func (b *ffmpegBuilder) AddInput(src *inputBuilder) {
	b.inputs = append(b.inputs, src)
}

func (b *ffmpegBuilder) AddOutput(dst *outputBuilder) {
	b.outputs = append(b.outputs, dst)
}

func (b *ffmpegBuilder) SetBinary(bin string) {
	b.binary = bin
}

func (b *ffmpegBuilder) SetFlag(flag string) {
	b.options[flag] = ""
}

func (b *ffmpegBuilder) SetOption(name, value string) {
	b.options[name] = value
}

func (b *ffmpegBuilder) Args() (args []string) {
	for name, val := range b.options {
		args = append(args, "-"+name)
		if val != "" {
			args = append(args, val)
		}
	}

	for _, input := range b.inputs {
		args = append(args, input.Args()...)
	}
	for _, output := range b.outputs {
		args = append(args, output.Args()...)
	}

	return
}

func (b *ffmpegBuilder) Command(ctx context.Context) *exec.Cmd {
	bin := "ffmpeg"
	if b.binary != "" {
		bin = b.binary
	}

	return exec.CommandContext(ctx, bin, b.Args()...)
}

// inputBuilder is the builder for ffmpeg input options
type inputBuilder struct {
	path    string
	options map[string][]string
}

func newInputBuilder(path string) *inputBuilder {
	return &inputBuilder{
		path:    path,
		options: make(map[string][]string),
	}
}

func (b *inputBuilder) AddOption(name, value string) {
	b.options[name] = append(b.options[name], value)
}

func (b *inputBuilder) Args() (args []string) {
	for name, values := range b.options {
		for _, val := range values {
			args = append(args, "-"+name, val)
		}
	}
	return append(args, "-i", b.path)
}

// outputBuilder is the builder for ffmpeg output options
type outputBuilder struct {
	path    string
	options map[string][]string
}

func newOutputBuilder(path string) *outputBuilder {
	return &outputBuilder{
		path:    path,
		options: make(map[string][]string),
	}
}

func (b *outputBuilder) AddOption(name, value string) {
	b.options[name] = append(b.options[name], value)
}

func (b *outputBuilder) Args() (args []string) {
	for name, values := range b.options {
		for _, val := range values {
			args = append(args, "-"+name, val)
		}
	}
	return append(args, b.path)
}

// AddMetadata is the shortcut for adding "metadata" option
func (b *outputBuilder) AddMetadata(stream, key, value string) {
	optVal := strings.TrimSpace(key) + "=" + strings.TrimSpace(value)

	if stream != "" {
		b.AddOption("metadata:"+stream, optVal)
	} else {
		b.AddOption("metadata", optVal)
	}
}
