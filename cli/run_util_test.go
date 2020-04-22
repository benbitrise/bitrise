package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/bitrise-io/bitrise/bitrise"
	"github.com/bitrise-io/bitrise/configs"
	"github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/pointers"
	"github.com/stretchr/testify/require"
)

func TestIsSecretFiltering(t *testing.T) {
	t.Log("flag influences the filtering")
	{
		set, err := isSecretFiltering(pointers.NewBoolPtr(true), []envmanModels.EnvironmentItemModel{})
		require.NoError(t, err)
		require.True(t, set)

		set, err = isSecretFiltering(pointers.NewBoolPtr(false), []envmanModels.EnvironmentItemModel{})
		require.NoError(t, err)
		require.False(t, set)
	}

	t.Log("secret influences the filtering")
	{
		set, err := isSecretFiltering(nil, []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "true"},
		})
		require.NoError(t, err)
		require.True(t, set)

		set, err = isSecretFiltering(nil, []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "false"},
		})
		require.NoError(t, err)
		require.False(t, set)
	}

	t.Log("flag has priority")
	{
		set, err := isSecretFiltering(pointers.NewBoolPtr(false), []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "true"},
		})
		require.NoError(t, err)
		require.False(t, set)
	}

	t.Log("secrets are exposed")
	{
		set, err := isSecretFiltering(nil, []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "true", "opts": map[string]interface{}{"is_expand": true}},
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "false", "opts": map[string]interface{}{"is_expand": true}},
		})
		require.NoError(t, err)
		require.False(t, set)

		set, err = isSecretFiltering(nil, []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "true", "opts": map[string]interface{}{"is_expand": true}},
			envmanModels.EnvironmentItemModel{"BITRISE_SECRET_FILTERING": "$BITRISE_SECRET_FILTERING", "opts": map[string]interface{}{"is_expand": true}},
		})
		require.NoError(t, err)
		require.True(t, set)
	}
}

func TestIsPRMode(t *testing.T) {
	prModeEnv := os.Getenv(configs.PRModeEnvKey)
	prIDEnv := os.Getenv(configs.PullRequestIDEnvKey)

	// cleanup Envs after these tests
	defer func() {
		require.NoError(t, os.Setenv(configs.PRModeEnvKey, prModeEnv))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, prIDEnv))
	}()

	t.Log("Should be false for: prGlobalFlag: nil, prModeEnv: '', prIDEnv: ''")
	{
		require.NoError(t, os.Setenv(configs.PRModeEnvKey, ""))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(nil, []envmanModels.EnvironmentItemModel{})
		require.NoError(t, err)
		require.Equal(t, false, pr)
	}

	t.Log("Should be false for: prGlobalFlag: nil, prModeEnv: '', prIDEnv: '', secrets: false")
	{
		inventoryStr := `
envs:
- PR: "false"
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, ""))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, false, pr)
	}

	t.Log("Should be false for: prGlobalFlag: nil, prModeEnv: 'false', prIDEnv: '', secrets: ''")
	{
		inventoryStr := `
envs:
- PR: ""
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, "false"))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, false, pr)
	}

	t.Log("Should be false for: prGlobalFlag: false, prModeEnv: 'true', prIDEnv: 'ID', secrets: 'true'")
	{
		inventoryStr := `
envs:
- PR: "true"
- PULL_REQUEST_ID: "ID"
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, "true"))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, "ID"))

		pr, err := isPRMode(pointers.NewBoolPtr(false), inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, false, pr)
	}

	t.Log("Should be true for: prGlobalFlag: true, prModeEnv: '', prIDEnv: ''")
	{
		require.NoError(t, os.Setenv(configs.PRModeEnvKey, ""))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(pointers.NewBoolPtr(true), []envmanModels.EnvironmentItemModel{})
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}

	t.Log("Should be true for: prGlobalFlag: true, prModeEnv: '', prIDEnv: '', secrets: false")
	{
		inventoryStr := `
envs:
- PR: "false"
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, ""))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(pointers.NewBoolPtr(true), inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}

	t.Log("Should be true for: prGlobalFlag: nil, prModeEnv: 'true', prIDEnv: '', secrets: false")
	{
		inventoryStr := `
envs:
- PR: "false"
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, "true"))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}

	t.Log("Should be true for: prGlobalFlag: nil, prModeEnv: 'false', prIDEnv: 'some', secrets: false")
	{
		inventoryStr := `
envs:
- PR: "false"
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, "false"))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, "some"))

		pr, err := isPRMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}

	t.Log("Should be true for: prGlobalFlag: nil, prModeEnv: '', prIDEnv: '', secrets: true")
	{
		inventoryStr := `
envs:
- PR: "true"
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, ""))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}

	t.Log("Should be true for: prGlobalFlag: nil, prModeEnv: 'false', prIDEnv: '', secrets: true")
	{
		inventoryStr := `
envs:
- PR: ""
- PULL_REQUEST_ID: "some"
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, "false"))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}

	t.Log("Should be true for: prGlobalFlag: true, prModeEnv: 'false', prIDEnv: '', secrets: false")
	{
		inventoryStr := `
envs:
- PR: "false"
- PULL_REQUEST_ID: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.PRModeEnvKey, "false"))
		require.NoError(t, os.Setenv(configs.PullRequestIDEnvKey, ""))

		pr, err := isPRMode(pointers.NewBoolPtr(true), inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, pr)
	}
}

func TestIsCIMode(t *testing.T) {
	ciModeEnv := os.Getenv(configs.CIModeEnvKey)

	defer func() {
		require.NoError(t, os.Setenv(configs.CIModeEnvKey, ciModeEnv))
	}()

	t.Log("Should be false for: ciGlobalFlag: nil, ciModeEnv: 'false'")
	{
		require.NoError(t, os.Setenv(configs.CIModeEnvKey, "false"))

		ci, err := isCIMode(nil, []envmanModels.EnvironmentItemModel{})
		require.NoError(t, err)
		require.Equal(t, false, ci)
	}

	t.Log("Should be false for: ciGlobalFlag: false, ciModeEnv: 'false' secrets: false")
	{
		inventoryStr := `
envs:
- CI: "false"
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.CIModeEnvKey, "false"))

		ci, err := isCIMode(pointers.NewBoolPtr(false), inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, false, ci)
	}

	t.Log("Should be true for: ciGlobalFlag: true, ciModeEnv: 'false'")
	{
		require.NoError(t, os.Setenv(configs.CIModeEnvKey, ""))

		ci, err := isCIMode(pointers.NewBoolPtr(true), []envmanModels.EnvironmentItemModel{})
		require.NoError(t, err)
		require.Equal(t, true, ci)
	}

	t.Log("Should be true for: ciGlobalFlag: true, ciModeEnv: '' secrets: false")
	{
		inventoryStr := `
envs:
- CI: "false"
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.CIModeEnvKey, ""))

		ci, err := isCIMode(pointers.NewBoolPtr(true), inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, ci)
	}

	t.Log("Should be true for: ciGlobalFlag: nil, ciModeEnv: 'true' secrets: false")
	{
		inventoryStr := `
envs:
- CI: ""
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.CIModeEnvKey, "true"))

		ci, err := isCIMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, ci)
	}

	t.Log("Should be true for: ciGlobalFlag: nil, ciModeEnv: '' secrets: true")
	{
		inventoryStr := `
envs:
- CI: "true"
`
		inventory, err := bitrise.InventoryModelFromYAMLBytes([]byte(inventoryStr))
		require.NoError(t, err)

		require.NoError(t, os.Setenv(configs.CIModeEnvKey, ""))

		ci, err := isCIMode(nil, inventory.Envs)
		require.NoError(t, err)
		require.Equal(t, true, ci)
	}
}

func TestExpandEnvs(t *testing.T) {
	configStr := `
format_version: 1.3.0
default_step_lib_source: "https://github.com/bitrise-io/bitrise-steplib.git"

workflows:
  test:
    envs:
    - ENV0: "Hello"
    - ENV1: "$ENV0 world"
    steps:
    - script:
        inputs:
        - content: |
            #!/bin/bash
            envman add --key ENV2 --value "$ENV1 !"
    - script:
        inputs:
        - content: |
            #!/bin/bash
            echo "ENV2: $ENV2"
            if [ "$ENV2" != "Hello world !" ] ; then
              echo "Actual ($ENV2), excpected (Hello world !)"
              exit 1
            fi
`

	require.NoError(t, configs.InitPaths())

	config, warnings, err := bitrise.ConfigModelFromYAMLBytes([]byte(configStr))
	require.NoError(t, err)
	require.Equal(t, 0, len(warnings))

	buildRunResults, err := runWorkflowWithConfiguration(time.Now(), "test", config, []envmanModels.EnvironmentItemModel{})
	require.NoError(t, err)
	require.Equal(t, 2, len(buildRunResults.SuccessSteps))
	require.Equal(t, 0, len(buildRunResults.FailedSteps))
	require.Equal(t, 0, len(buildRunResults.FailedSkippableSteps))
	require.Equal(t, 0, len(buildRunResults.SkippedSteps))
}

func TestEvaluateInputs(t *testing.T) {
	configStr := `
format_version: 1.3.0
default_step_lib_source: "https://github.com/bitrise-io/bitrise-steplib.git"

workflows:
  test:
    envs:
    - TEST_KEY: "test value"
    steps:
    - script:
        title: "Template test"
        inputs:
        - content: |
            #!/bin/bash
            set -v
            {{if .IsCI}}
            exit 1
            {{else}}
            exit 0
            {{end}}
          opts:
            is_template: true
    - script:
        title: "Template test"
        inputs:
        - content: |
            #!/bin/bash
            set -v
            {{if enveq "TEST_KEY" "test value"}}
            exit 0
            {{else}}
            exit 1
            {{end}}
          opts:
            is_template: true
`

	require.NoError(t, configs.InitPaths())

	config, warnings, err := bitrise.ConfigModelFromYAMLBytes([]byte(configStr))
	require.NoError(t, err)
	require.Equal(t, 0, len(warnings))

	buildRunResults, err := runWorkflowWithConfiguration(time.Now(), "test", config, []envmanModels.EnvironmentItemModel{})
	require.Equal(t, nil, err)
	require.Equal(t, 0, len(buildRunResults.SkippedSteps))
	require.Equal(t, 2, len(buildRunResults.SuccessSteps))
	require.Equal(t, 0, len(buildRunResults.FailedSteps))
	require.Equal(t, 0, len(buildRunResults.FailedSkippableSteps))
}

func TestGetBitriseConfigFromBase64Data(t *testing.T) {
	configStr := `
format_version: 0.9.10
default_step_lib_source: "https://github.com/bitrise-io/bitrise-steplib.git"

workflows:
  target:
    title: target
`
	configBytes := []byte(configStr)
	configBase64Str := base64.StdEncoding.EncodeToString(configBytes)

	config, warnings, err := GetBitriseConfigFromBase64Data(configBase64Str)
	require.NoError(t, err)
	require.Equal(t, 0, len(warnings))

	require.Equal(t, "0.9.10", config.FormatVersion)
	require.Equal(t, "https://github.com/bitrise-io/bitrise-steplib.git", config.DefaultStepLibSource)

	workflow, found := config.Workflows["target"]
	require.Equal(t, true, found)
	require.Equal(t, "target", workflow.Title)
}

func TestGetInventoryFromBase64Data(t *testing.T) {
	inventoryStr := `
envs:
  - MY_HOME: $HOME
    opts:
      is_expand: true
`
	inventoryBytes := []byte(inventoryStr)
	inventoryBase64Str := base64.StdEncoding.EncodeToString(inventoryBytes)

	inventory, err := GetInventoryFromBase64Data(inventoryBase64Str)
	require.NoError(t, err)

	env := inventory[0]

	key, value, err := env.GetKeyValuePair()
	require.NoError(t, err)
	require.Equal(t, "MY_HOME", key)
	require.Equal(t, "$HOME", value)

	opts, err := env.GetOptions()
	require.NoError(t, err)
	require.Equal(t, true, *opts.IsExpand)
}

func TestInvalidStepID(t *testing.T) {
	configStr := `
format_version: 1.3.0
default_step_lib_source: "https://github.com/bitrise-io/bitrise-steplib.git"

workflows:
  target:
    title: Invalid step id
    steps:
    - invalid-step:
    - invalid-step:
    - invalid-step:
`

	require.NoError(t, configs.InitPaths())

	config, warnings, err := bitrise.ConfigModelFromYAMLBytes([]byte(configStr))
	require.NoError(t, err)
	require.Equal(t, 0, len(warnings))

	results, err := runWorkflowWithConfiguration(time.Now(), "target", config, []envmanModels.EnvironmentItemModel{})
	require.Equal(t, 1, len(results.StepmanUpdates))
}

func TestAddTestMetadata(t *testing.T) {
	t.Log("test empty dir")
	{
		testDirPath, err := pathutil.NormalizedOSTempDirPath("testing")
		if err != nil {
			t.Fatalf("failed to create testing dir, error: %s", err)
		}

		testResultStepInfo := models.TestResultStepInfo{}

		exists, err := pathutil.IsDirExists(testDirPath)
		if err != nil {
			t.Fatalf("failed to check if dir exists, error: %s", err)
		}

		if !exists {
			t.Fatal("test dir should exits")
		}

		if err := addTestMetadata(testDirPath, testResultStepInfo); err != nil {
			t.Fatalf("failed to normalize test dir, error: %s", err)
		}

		exists, err = pathutil.IsDirExists(testDirPath)
		if err != nil {
			t.Fatalf("failed to check if dir exists, error: %s", err)
		}

		if exists {
			t.Fatal("test dir should not exits")
		}
	}

	t.Log("test not empty dir")
	{
		testDirPath, err := pathutil.NormalizedOSTempDirPath("testing")
		if err != nil {
			t.Fatalf("failed to create testing dir, error: %s", err)
		}

		testResultStepInfo := models.TestResultStepInfo{}

		exists, err := pathutil.IsDirExists(testDirPath)
		if err != nil {
			t.Fatalf("failed to check if dir exists, error: %s", err)
		}

		if !exists {
			t.Fatal("test dir should exits")
		}

		if err := fileutil.WriteStringToFile(filepath.Join(testDirPath, "test-file"), "test-content"); err != nil {
			t.Fatalf("failed to write file, error: %s", err)
		}

		if err := addTestMetadata(testDirPath, testResultStepInfo); err != nil {
			t.Fatalf("failed to normalize test dir, error: %s", err)
		}

		exists, err = pathutil.IsDirExists(testDirPath)
		if err != nil {
			t.Fatalf("failed to check if dir exists, error: %s", err)
		}

		if !exists {
			t.Fatal("test dir should exits")
		}

		exists, err = pathutil.IsPathExists(filepath.Join(testDirPath, "test-file"))
		if err != nil {
			t.Fatalf("failed to check if dir exists, error: %s", err)
		}

		if !exists {
			t.Fatal("test file should exits")
		}

		exists, err = pathutil.IsPathExists(filepath.Join(testDirPath, "step-info.json"))
		if err != nil {
			t.Fatalf("failed to check if dir exists, error: %s", err)
		}

		if !exists {
			t.Fatal("step-info.json file should exits")
		}
	}
}

func Test_activateStepLibStep(t *testing.T) {
	tests := []struct {
		name        string
		stepIDData  models.StepIDData
		wantVersion string
		wantErr     bool
	}{
		{
			name: "Major version lock",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "1",
			},
			wantVersion: "1.10.1",
			wantErr:     false,
		},
		{
			name: "Major version lock (long form)",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "1.x.x",
			},
			wantVersion: "1.10.1",
			wantErr:     false,
		},
		{
			name: "Minor version lock",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "2.3",
			},
			wantVersion: "2.3.7",
			wantErr:     false,
		},
		{
			name: "Minor version lock (long form)",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "2.3.x",
			},
			wantVersion: "2.3.7",
			wantErr:     false,
		},
		{
			name: "Patch version lock",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "2.3.2",
			},
			wantVersion: "2.3.2",
			wantErr:     false,
		},
		{
			name: "Invalid version lock",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "1.2.3.4",
			},
			wantVersion: "",
			wantErr:     true,
		},
		{
			name: "Latest version (not supported at the moment)",
			stepIDData: models.StepIDData{
				SteplibSource: "https://github.com/bitrise-io/bitrise-steplib.git",
				IDorURI:       "xcode-archive",
				Version:       "x.x.x",
			},
			wantVersion: "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := pathutil.NormalizedOSTempDirPath("activateStepLibStep")
			if err != nil {
				t.Errorf("failed to create tmp dir: %s", err)
			}

			destination := tmpDir
			stepYMLCopyPth := filepath.Join(tmpDir, "step-yml", "step.yml")

			if err := os.MkdirAll(filepath.Dir(stepYMLCopyPth), 0777); err != nil {
				t.Errorf("failed to create dir for step.yml: %s", err)
			}

			got, _, err := activateStepLibStep(tt.stepIDData, destination, stepYMLCopyPth, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("activateStepLibStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Version != tt.wantVersion {
				t.Errorf("activateStepLibStep() got = %v, want %v", got.Version, tt.wantVersion)
			}

			if err != nil {
				return
			}
			if exist, err := pathutil.IsPathExists(filepath.Join(destination, "step.yml")); err != nil || !exist {
				t.Errorf("step not activate at: %s", destination)
			}

			if exist, err := pathutil.IsPathExists(stepYMLCopyPth); err != nil || !exist {
				t.Errorf("step.yml not copied at: %s", stepYMLCopyPth)
			}
		})
	}
}

func TestExpandStepInputs(t *testing.T) {
	tests := []struct {
		name   string
		envs   []envmanModels.EnvironmentItemModel
		inputs []envmanModels.EnvironmentItemModel
		want   map[string]string
	}{
		{
			name: "Env does not depend on input",
			envs: []envmanModels.EnvironmentItemModel{
				{"simulator_device": "$simulator_major", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_major": "12", "opts": map[string]interface{}{"is_sensitive": false}},
				{"simulator_os_version": "$simulator_device", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_major":      "12",
				"simulator_os_version": "",
			},
		},
		{
			name: "Env does not depend on input (input order switched)",
			envs: []envmanModels.EnvironmentItemModel{
				{"simulator_device": "$simulator_major", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "$simulator_device", "opts": map[string]interface{}{"is_sensitive": false}},
				{"simulator_major": "12", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_major":      "12",
				"simulator_os_version": "",
			},
		},
		{
			name: "Secrets are removed",
			envs: []envmanModels.EnvironmentItemModel{},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "13.3", "opts": map[string]interface{}{"is_sensitive": false}},
				{"simulator_device": "iPhone 8 Plus", "opts": map[string]interface{}{"is_sensitive": false}},
				{"secret_input": "top secret", "opts": map[string]interface{}{"is_sensitive": true}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
				"simulator_device":     "iPhone 8 Plus",
				// "secret_input":         "",
			},
		},
		{
			name: "No env expansion, missing options (sensive input).",
			envs: []envmanModels.EnvironmentItemModel{},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "13.3", "opts": map[string]interface{}{}},
				{"simulator_device": "iPhone 8 Plus", "opts": map[string]interface{}{}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
				"simulator_device":     "iPhone 8 Plus",
			},
		},
		{
			name: "No env expansion, options specified",
			envs: []envmanModels.EnvironmentItemModel{},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "13.3", "opts": map[string]interface{}{"is_sensitive": false}},
				{"simulator_device": "iPhone 8 Plus", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
				"simulator_device":     "iPhone 8 Plus",
			},
		},
		{
			name: "Env expansion, input contains env var.",
			envs: []envmanModels.EnvironmentItemModel{
				{"SIMULATOR_OS_VERSION": "13.3", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "$SIMULATOR_OS_VERSION", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
			},
		},
		{
			name: "Env var expansion, input expansion",
			envs: []envmanModels.EnvironmentItemModel{
				{"SIMULATOR_OS_MAJOR_VERSION": "13", "opts": map[string]interface{}{"is_sensitive": false}},
				{"SIMULATOR_OS_MINOR_VERSION": "3", "opts": map[string]interface{}{"is_sensitive": false}},
				{"SIMULATOR_OS_VERSION": "$SIMULATOR_OS_MAJOR_VERSION.$SIMULATOR_OS_MINOR_VERSION", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "$SIMULATOR_OS_VERSION", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
			},
		},
		{
			name: "Input expansion, input refers other input",
			envs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "12.1", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "13.3", "opts": map[string]interface{}{"is_sensitive": false}},
				{"simulator_device": "iPhone 8 ($simulator_os_version)", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
				"simulator_device":     "iPhone 8 (13.3)",
			},
		},
		{
			name: "Input expansion, input can not refer other input declared after it",
			envs: []envmanModels.EnvironmentItemModel{
				{"simulator_os_version": "12.1", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"simulator_device": "iPhone 8 ($simulator_os_version)", "opts": map[string]interface{}{"is_sensitive": false}},
				{"simulator_os_version": "13.3", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"simulator_os_version": "13.3",
				"simulator_device":     "iPhone 8 (12.1)",
			},
		},
		{
			name: "Input refers itself, env refers itself",
			envs: []envmanModels.EnvironmentItemModel{
				{"ENV_LOOP": "$ENV_LOOP", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"loop": "$loop", "opts": map[string]interface{}{"is_sensitive": false}},
				{"env_loop": "$ENV_LOOP", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"loop":     "",
				"env_loop": "",
			},
		},
		{
			name: "Input refers itself, env refers itself; both have prefix included",
			envs: []envmanModels.EnvironmentItemModel{
				{"ENV_LOOP": "Env Something: $ENV_LOOP", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"loop": "Something: $loop", "opts": map[string]interface{}{"is_sensitive": false}},
				{"env_loop": "$ENV_LOOP", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"loop":     "Something: ",
				"env_loop": "Env Something: ",
			},
		},
		{
			name: "Inputs refer inputs in a chain, with prefix included",
			envs: []envmanModels.EnvironmentItemModel{},
			inputs: []envmanModels.EnvironmentItemModel{
				{"similar2": "anything", "opts": map[string]interface{}{"is_sensitive": false}},
				{"similar": "$similar2", "opts": map[string]interface{}{"is_sensitive": false}},
				{"env": "Something: $similar", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"similar2": "anything",
				"similar":  "anything",
				"env":      "Something: anything",
			},
		},
		{
			name: "References in a loop are not expanded",
			envs: []envmanModels.EnvironmentItemModel{
				{"B": "$A", "opts": map[string]interface{}{"is_sensitive": false}},
				{"A": "$B", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			inputs: []envmanModels.EnvironmentItemModel{
				{"a": "$b", "opts": map[string]interface{}{"is_sensitive": false}},
				{"b": "$c", "opts": map[string]interface{}{"is_sensitive": false}},
				{"c": "$a", "opts": map[string]interface{}{"is_sensitive": false}},
				{"env": "$A", "opts": map[string]interface{}{"is_sensitive": false}},
			},
			want: map[string]string{
				"a":   "",
				"b":   "",
				"c":   "",
				"env": "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := expandStepInputs(test.inputs, test.envs)

			require.NotNil(t, got)
			if !reflect.DeepEqual(test.want, got) {
				t.Fatalf("expandStepInputs() actual: %v expected: %v", got, test.want)
			}
		})
	}
}
