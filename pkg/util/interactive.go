package util

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

func Select(question string, items []string) (string, error) {
	answer := ""
	prompt := &survey.Select{
		Message: question,
		Options: items,
	}
	err := survey.AskOne(prompt, &answer)

	return answer, err
}

func Ask(question, defaultInput string) (string, error) {
	answer := ""
	prompt := &survey.Input{
		Message: question,
		Default: defaultInput,
	}
	err := survey.AskOne(prompt, &answer)

	return answer, err
}

func AskPath(question string, value string, validate survey.Validator) (string, error) {
	answer := ""

	err := survey.AskOne(&survey.Input{
		Message: question,
		Default: value,
		Suggest: suggestPaths,
	}, &answer, withValidator(validate))

	return answer, err
}

func IsDirectoryValidator(input interface{}) error {
	fileInfo, err := os.Stat(input.(string))
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return errors.New("path has to be a directory")
	}

	return nil
}

func withValidator(validate survey.Validator) survey.AskOpt {
	if validate == nil {
		return nil
	}

	return survey.WithValidator(validate)
}

func suggestPaths(toComplete string) []string {
	files, _ := filepath.Glob(toComplete + "*")
	return files
}
