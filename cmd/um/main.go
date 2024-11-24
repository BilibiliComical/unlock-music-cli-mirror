package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"
	"unlock-music.dev/cli/algo/common"
	_ "unlock-music.dev/cli/algo/kgm"
	_ "unlock-music.dev/cli/algo/kwm"
	_ "unlock-music.dev/cli/algo/ncm"
	"unlock-music.dev/cli/algo/qmc"
	_ "unlock-music.dev/cli/algo/tm"
	_ "unlock-music.dev/cli/algo/xiami"
	_ "unlock-music.dev/cli/algo/ximalaya"
	"unlock-music.dev/cli/internal/ffmpeg"
	"unlock-music.dev/cli/internal/sniff"
	"unlock-music.dev/cli/internal/utils"
)

var AppVersion = "v0.2.11"

var logger = setupLogger(false) // TODO: inject logger to application, instead of using global logger

func main() {
	module, ok := debug.ReadBuildInfo()
	if ok && module.Main.Version != "(devel)" {
		AppVersion = module.Main.Version
	}
	app := cli.App{
		Name:     "Unlock Music CLI",
		HelpName: "um",
		Usage:    "Unlock your encrypted music file https://git.unlock-music.dev/um/cli",
		Version:  fmt.Sprintf("%s (%s,%s/%s)", AppVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "input", Aliases: []string{"i"}, Usage: "path to input file or dir", Required: false},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "path to output dir", Required: false},
			&cli.StringFlag{Name: "qmc-mmkv", Aliases: []string{"db"}, Usage: "path to qmc mmkv (.crc file also required)", Required: false},
			&cli.StringFlag{Name: "qmc-mmkv-key", Aliases: []string{"key"}, Usage: "mmkv password (16 ascii chars)", Required: false},
			&cli.BoolFlag{Name: "remove-source", Aliases: []string{"rs"}, Usage: "remove source file", Required: false, Value: false},
			&cli.BoolFlag{Name: "skip-noop", Aliases: []string{"n"}, Usage: "skip noop decoder", Required: false, Value: true},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"V"}, Usage: "verbose logging", Required: false, Value: false},
			&cli.BoolFlag{Name: "update-metadata", Usage: "update metadata & album art from network", Required: false, Value: false},
			&cli.BoolFlag{Name: "overwrite", Usage: "overwrite output file without asking", Required: false, Value: false},
			&cli.BoolFlag{Name: "watch", Usage: "watch the input dir and process new files", Required: false, Value: false},

			&cli.BoolFlag{Name: "supported-ext", Usage: "show supported file extensions and exit", Required: false, Value: false},
		},

		Action:          appMain,
		Copyright:       fmt.Sprintf("Copyright (c) 2020 - %d Unlock Music https://git.unlock-music.dev/um/cli/src/branch/master/LICENSE", time.Now().Year()),
		HideHelpCommand: true,
		UsageText:       "um [-o /path/to/output/dir] [--extra-flags] [-i] /path/to/input",
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal("run app failed", zap.Error(err))
	}
}

func printSupportedExtensions() {
	var exts []string
	extSet := make(map[string]int)
	for _, factory := range common.DecoderRegistry {
		ext := strings.TrimPrefix(factory.Suffix, ".")
		if n, ok := extSet[ext]; ok {
			extSet[ext] = n + 1
		} else {
			extSet[ext] = 1
		}
	}
	for ext := range extSet {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	for _, ext := range exts {
		fmt.Printf("%s: %d\n", ext, extSet[ext])
	}
}

func setupLogger(verbose bool) *zap.Logger {
	logConfig := zap.NewProductionEncoderConfig()
	logConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	enabler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		if verbose {
			return true
		}
		return level >= zapcore.InfoLevel
	})

	return zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(logConfig),
		os.Stdout,
		enabler,
	))
}

func appMain(c *cli.Context) (err error) {
	logger = setupLogger(c.Bool("verbose"))

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if c.Bool("supported-ext") {
		printSupportedExtensions()
		return nil
	}
	input := c.String("input")
	if input == "" {
		switch c.Args().Len() {
		case 0:
			input = cwd
		case 1:
			input = c.Args().Get(0)
		default:
			return errors.New("please specify input file (or directory)")
		}
	}

	input, absErr := filepath.Abs(input)
	if absErr != nil {
		return fmt.Errorf("get abs path failed: %w", absErr)
	}

	output := c.String("output")
	inputStat, err := os.Stat(input)
	if err != nil {
		return err
	}

	var inputDir string
	if inputStat.IsDir() {
		inputDir = input
	} else {
		inputDir = filepath.Dir(input)
	}
	inputDir, absErr = filepath.Abs(inputDir)
	if absErr != nil {
		return fmt.Errorf("get abs path (inputDir) failed: %w", absErr)
	}

	if output == "" {
		// Default to where the input dir is
		output = inputDir
	}
	logger.Debug("resolve input/output path", zap.String("inputDir", inputDir), zap.String("input", input), zap.String("output", output))

	outputStat, err := os.Stat(output)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(output, 0755)
		}
		if err != nil {
			return err
		}
	} else if !outputStat.IsDir() {
		return errors.New("output should be a writable directory")
	}

	if mmkv := c.String("qmc-mmkv"); mmkv != "" {
		// If key is not set, the mmkv vault will be treated as unencrypted.
		key := c.String("qmc-mmkv-key")
		err := qmc.OpenMMKV(mmkv, key, logger)
		if err != nil {
			return err
		}
	}

	proc := &processor{
		logger:          logger,
		inputDir:        inputDir,
		outputDir:       output,
		skipNoopDecoder: c.Bool("skip-noop"),
		removeSource:    c.Bool("remove-source"),
		updateMetadata:  c.Bool("update-metadata"),
		overwriteOutput: c.Bool("overwrite"),
	}

	if inputStat.IsDir() {
		watchDir := c.Bool("watch")
		if !watchDir {
			return proc.processDir(input)
		} else {
			return proc.watchDir(input)
		}
	} else {
		return proc.processFile(input)
	}

}

type processor struct {
	logger    *zap.Logger
	inputDir  string
	outputDir string

	skipNoopDecoder bool
	removeSource    bool
	updateMetadata  bool
	overwriteOutput bool
}

func (p *processor) watchDir(inputDir string) error {
	if err := p.processDir(inputDir); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
					// try open with exclusive mode, to avoid file is still writing
					f, err := os.OpenFile(event.Name, os.O_RDONLY, os.ModeExclusive)
					if err != nil {
						logger.Debug("failed to open file exclusively", zap.String("path", event.Name), zap.Error(err))
						time.Sleep(1 * time.Second) // wait for file writing complete
						continue
					}
					_ = f.Close()

					if err := p.processFile(event.Name); err != nil {
						logger.Warn("failed to process file", zap.String("path", event.Name), zap.Error(err))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("file watcher got error", zap.Error(err))
			}
		}
	}()

	err = watcher.Add(inputDir)
	if err != nil {
		return fmt.Errorf("failed to watch dir %s: %w", inputDir, err)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	<-signalCtx.Done()
	return nil
}

func (p *processor) processDir(inputDir string) error {
	items, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}

	var lastError error = nil
	for _, item := range items {
		filePath := filepath.Join(inputDir, item.Name())
		if item.IsDir() {
			if err = p.processDir(filePath); err != nil {
				lastError = err
			}
			continue
		}

		if err := p.processFile(filePath); err != nil {
			lastError = err
			logger.Error("conversion failed", zap.String("source", item.Name()), zap.Error(err))
		}
	}
	if lastError != nil {
		return fmt.Errorf("last error: %w", lastError)
	}
	return nil
}

func (p *processor) processFile(filePath string) error {
	p.logger.Debug("processFile", zap.String("file", filePath), zap.String("inputDir", p.inputDir))

	allDec := common.GetDecoder(filePath, p.skipNoopDecoder)
	if len(allDec) == 0 {
		return errors.New("skipping while no suitable decoder")
	}

	if err := p.process(filePath, allDec); err != nil {
		return err
	}

	// if source file need to be removed
	if p.removeSource {
		err := os.RemoveAll(filePath)
		if err != nil {
			return err
		}
		logger.Info("source file removed after success conversion", zap.String("source", filePath))
	}
	return nil
}

func (p *processor) findDecoder(decoders []common.DecoderFactory, params *common.DecoderParams) (*common.Decoder, *common.DecoderFactory, error) {
	for _, factory := range decoders {
		dec := factory.Create(params)
		err := dec.Validate()
		if err == nil {
			return &dec, &factory, nil
		}
		logger.Warn("try decode failed", zap.Error(err))
	}
	return nil, nil, errors.New("no any decoder can resolve the file")
}

func (p *processor) process(inputFile string, allDec []common.DecoderFactory) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	logger := logger.With(zap.String("source", inputFile))

	pDec, decoderFactory, err := p.findDecoder(allDec, &common.DecoderParams{
		Reader:    file,
		Extension: filepath.Ext(inputFile),
		FilePath:  inputFile,
		Logger:    logger,
	})
	if err != nil {
		return err
	}
	dec := *pDec

	params := &ffmpeg.UpdateMetadataParams{}

	header := bytes.NewBuffer(nil)
	_, err = io.CopyN(header, dec, 64)
	if err != nil {
		return fmt.Errorf("read header failed: %w", err)
	}
	audio := io.MultiReader(header, dec)
	params.AudioExt = sniff.AudioExtensionWithFallback(header.Bytes(), ".mp3")

	if p.updateMetadata {
		if audioMetaGetter, ok := dec.(common.AudioMetaGetter); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// since ffmpeg doesn't support multiple input streams,
			// we need to write the audio to a temp file.
			// since qmc decoder doesn't support seeking & relying on ffmpeg probe, we need to read the whole file.
			// TODO: support seeking or using pipe for qmc decoder.
			params.Audio, err = utils.WriteTempFile(audio, params.AudioExt)
			if err != nil {
				return fmt.Errorf("updateAudioMeta write temp file: %w", err)
			}
			defer os.Remove(params.Audio)

			params.Meta, err = audioMetaGetter.GetAudioMeta(ctx)
			if err != nil {
				logger.Warn("get audio meta failed", zap.Error(err))
			}

			if params.Meta == nil { // reset audio meta if failed
				audio, err = os.Open(params.Audio)
				if err != nil {
					return fmt.Errorf("updateAudioMeta open temp file: %w", err)
				}
			}
		}
	}

	if p.updateMetadata && params.Meta != nil {
		if coverGetter, ok := dec.(common.CoverImageGetter); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if cover, err := coverGetter.GetCoverImage(ctx); err != nil {
				logger.Warn("get cover image failed", zap.Error(err))
			} else if imgExt, ok := sniff.ImageExtension(cover); !ok {
				logger.Warn("sniff cover image type failed", zap.Error(err))
			} else {
				params.AlbumArtExt = imgExt
				params.AlbumArt = cover
			}
		}
	}

	inputRelDir, err := filepath.Rel(p.inputDir, filepath.Dir(inputFile))
	if err != nil {
		return fmt.Errorf("get relative dir failed: %w", err)
	}

	inFilename := strings.TrimSuffix(filepath.Base(inputFile), decoderFactory.Suffix)
	outPath := filepath.Join(p.outputDir, inputRelDir, inFilename+params.AudioExt)

	if !p.overwriteOutput {
		_, err := os.Stat(outPath)
		if err == nil {
			logger.Warn("output file already exist, skip", zap.String("destination", outPath))
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat output file failed: %w", err)
		}
	}

	if params.Meta == nil {
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, audio); err != nil {
			return err
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if err := ffmpeg.UpdateMeta(ctx, outPath, params, logger); err != nil {
			return err
		}
	}

	logger.Info("successfully converted", zap.String("source", inputFile), zap.String("destination", outPath))
	return nil
}
