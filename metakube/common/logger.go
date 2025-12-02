package common

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/go-cty/cty"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerError struct {
	Message    string
	Attributes []string // Attribute names for error attribution
}

func (e *LoggerError) Error() string {
	return e.Message
}

func NewLoggerError(message string, attributes ...string) *LoggerError {
	return &LoggerError{
		Message:    message,
		Attributes: attributes,
	}
}

func NewLogger(config MetakubeProviderConfig, fd *os.File) (*zap.SugaredLogger, error) {
	var (
		ec    zapcore.EncoderConfig
		cores []zapcore.Core
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	)

	logDev := config.Development.ValueBool()
	logDebug := config.Debug.ValueBool()
	logPath := config.LogPath.ValueString()

	if logDev || logDebug {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if logDev {
		ec = zap.NewDevelopmentEncoderConfig()
		ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		ec = zap.NewProductionEncoderConfig()
		ec.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("[" + level.CapitalString() + "]")
		}
	}
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	ec.EncodeDuration = zapcore.StringDurationEncoder

	if logPath != "" {
		jsonEC := ec
		jsonEC.EncodeLevel = zapcore.LowercaseLevelEncoder
		sink, _, err := zap.Open(logPath)
		if err != nil {
			return nil, NewLoggerError(
				fmt.Sprintf("cannot access log location: %v", err),
				"log_path",
			)
		}
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(jsonEC), sink, level))
	}

	cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.AddSync(fd), level))
	core := zapcore.NewTee(cores...)
	return zap.New(core).Sugar(), nil
}

func LoggerToFrameworkDiagnostics(err error) fwdiag.Diagnostics {
	if err == nil {
		return nil
	}

	var diags fwdiag.Diagnostics
	var logErr *LoggerError
	if errors.As(err, &logErr) {
		for _, attr := range logErr.Attributes {
			diags.AddAttributeError(
				path.Root(attr),
				"Logger Configuration Error",
				logErr.Message,
			)
		}
		// If no specific attributes, add as general error
		if len(logErr.Attributes) == 0 {
			diags.AddError("Logger Configuration Error", logErr.Message)
		}
	} else {
		diags.AddError("Logger Configuration Error", err.Error())
	}

	return diags
}

func LoggerToSDKDiagnostics(err error) sdkdiag.Diagnostics {
	if err == nil {
		return nil
	}

	var logErr *LoggerError
	if errors.As(err, &logErr) {
		var attrPath cty.Path
		for _, attr := range logErr.Attributes {
			attrPath = append(attrPath, cty.GetAttrStep{Name: attr})
		}
		return sdkdiag.Diagnostics{{
			Severity:      sdkdiag.Error,
			Summary:       logErr.Message,
			AttributePath: attrPath,
		}}
	}

	return sdkdiag.Diagnostics{{
		Severity: sdkdiag.Error,
		Summary:  err.Error(),
	}}
}
