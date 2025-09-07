package config

type Config struct {
	Input   []InputConfig  `yaml:"inputs"`
	Filters []FilterRule   `yaml:"filters"`
	Output  []OutputConfig `yaml:"output"`
}

// inputs:
//   - type: file
//     path: /var/log/app.log
//     tag: application
//     format: json
type InputConfig struct {
	Type    string `yaml:"type"`
	Path    string `yaml:"path"`
	Tag     string `yaml:"tag"`
	Format  string `yaml:"format"`
	Address string `yaml:"address"`
}

// outputs:
//   - type: stdout
//     tag: ""
type OutputConfig struct {
	Type        string `yaml:"type"`
	Path        string `yaml:"path"`
	Tag         string `yaml:"tag"`
	Address     string `yaml:"address"`
	Compression bool   `yaml:"compression"`
}

// filters:
//   - type: execlude
//     tag: application
//     match:
//     execlude: "in ERROR,WARN"
//   - type: match
//     tag: application
//     match:
//     response_time: ">1.0"
//   - type: match
//     tag: network
//     match:
//     message: "~error|failed|critical"
type FilterRule struct {
	Type    string `yaml:"type"`
	Tag     string `yaml:"tag"`
	Pattern string `yaml:"pattern"`
	// Exclude map[string]string `yaml:"exclude"`
}

// type Filter struct {
// 	Type    string            `yaml:"type"`
// 	Tag     string            `yaml:"tag"`
// 	Match   map[string]string `yaml:"match"`
// 	Exclude map[string]string `yaml:"exclude"`
// }
