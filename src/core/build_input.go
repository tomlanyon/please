// Implementation of the BuildInput interface for simple cases of files in the local package.

package core

import (
	"path"
	"strings"
)

// A BuildInput represents some kind of input to a build rule. They can be implemented
// as either a file (in the local package or on the system) or another build rule.
type BuildInput interface {
	// Paths returns a slice of paths to the files of this input.
	Paths(graph *BuildGraph) []string
	// FullPaths is like Paths but includes the leading plz-out/gen directory.
	FullPaths(graph *BuildGraph) []string
	// LocalPaths returns paths within the local package
	LocalPaths(graph *BuildGraph) []string
	// Label returns the build label associated with this input, or nil if it doesn't have one (eg. it's just a file).
	Label() *BuildLabel
	// nonOutputLabel returns the build label associated with this input, or nil if it doesn't have
	// one or is a specific output of a rule.
	// This is fiddly enough that we don't want to expose it outside the package right now.
	nonOutputLabel() *BuildLabel
	// String returns a string representation of this input
	String() string
}

// FileLabel represents a file in the current package which is directly used by a target.
type FileLabel struct {
	// Name of the file
	File string
	// Name of the package
	Package string
}

// Paths returns a slice of paths to the files of this input.
func (label FileLabel) Paths(graph *BuildGraph) []string {
	return []string{path.Join(label.Package, label.File)}
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label FileLabel) FullPaths(graph *BuildGraph) []string {
	return label.Paths(graph)
}

// LocalPaths returns paths within the local package
func (label FileLabel) LocalPaths(graph *BuildGraph) []string {
	return []string{label.File}
}

// Label returns the build rule associated with this input. For a FileLabel it's always nil.
func (label FileLabel) Label() *BuildLabel {
	return nil
}

func (label FileLabel) nonOutputLabel() *BuildLabel {
	return nil
}

// String returns a string representation of this input.
func (label FileLabel) String() string {
	return label.File
}

// SystemFileLabel represents an absolute system dependency, which is not managed by the build system.
type SystemFileLabel struct {
	Path string
}

// Paths returns a slice of paths to the files of this input.
func (label SystemFileLabel) Paths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label SystemFileLabel) FullPaths(graph *BuildGraph) []string {
	return []string{ExpandHomePath(label.Path)}
}

// LocalPaths returns paths within the local package
func (label SystemFileLabel) LocalPaths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// Label returns the build rule associated with this input. For a SystemFileLabel it's always nil.
func (label SystemFileLabel) Label() *BuildLabel {
	return nil
}

func (label SystemFileLabel) nonOutputLabel() *BuildLabel {
	return nil
}

// String returns a string representation of this input.
func (label SystemFileLabel) String() string {
	return label.Path
}

// SystemPathLabel represents system dependency somewhere on PATH, which is not managed by the build system.
type SystemPathLabel struct {
	Name string
	Path []string
}

// Paths returns a slice of paths to the files of this input.
func (label SystemPathLabel) Paths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label SystemPathLabel) FullPaths(graph *BuildGraph) []string {
	// non-specified paths like "bash" are turned into absolute ones based on plz's PATH.
	// awkwardly this means we can't use the builtin exec.LookPath because the current
	// environment variable isn't necessarily the same as what's in our config.
	tool, err := LookPath(label.Name, label.Path)
	if err != nil {
		// This is a bit awkward, we can't signal an error here sensibly.
		panic(err)
	}
	return []string{tool}
}

// LocalPaths returns paths within the local package
func (label SystemPathLabel) LocalPaths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// Label returns the build rule associated with this input. For a SystemPathLabel it's always nil.
func (label SystemPathLabel) Label() *BuildLabel {
	return nil
}

func (label SystemPathLabel) nonOutputLabel() *BuildLabel {
	return nil
}

// String returns a string representation of this input.
func (label SystemPathLabel) String() string {
	return label.Name
}

// NamedOutputLabel represents a reference to a subset of named outputs from a rule.
// The rule must have declared those as a named group.
type NamedOutputLabel struct {
	BuildLabel
	Output string
}

// Paths returns a slice of paths to the files of this input.
func (label NamedOutputLabel) Paths(graph *BuildGraph) []string {
	return addPathPrefix(graph.TargetOrDie(label.BuildLabel).NamedOutputs(label.Output), label.PackageName)
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label NamedOutputLabel) FullPaths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label.BuildLabel)
	return addPathPrefix(target.NamedOutputs(label.Output), target.OutDir())
}

// LocalPaths returns paths within the local package
func (label NamedOutputLabel) LocalPaths(graph *BuildGraph) []string {
	return graph.TargetOrDie(label.BuildLabel).NamedOutputs(label.Output)
}

// Label returns the build rule associated with this input. For a NamedOutputLabel it's always non-nil.
func (label NamedOutputLabel) Label() *BuildLabel {
	return &label.BuildLabel
}

func (label NamedOutputLabel) nonOutputLabel() *BuildLabel {
	return nil
}

// String returns a string representation of this input.
func (label NamedOutputLabel) String() string {
	return label.BuildLabel.String() + "|" + label.Output
}

// TryParseNamedOutputLabel attempts to parse a build output label. It's allowed to just be
// a normal build label as well.
// The syntax is an extension of normal build labels: //package:target|output
func TryParseNamedOutputLabel(target, currentPath string) (BuildInput, error) {
	if index := strings.IndexRune(target, '|'); index != -1 && index != len(target)-1 {
		label, err := TryParseBuildLabel(target[:index], currentPath)
		return NamedOutputLabel{BuildLabel: label, Output: target[index+1:]}, err
	}
	return TryParseBuildLabel(target, currentPath)
}

// MustParseNamedOutputLabel is like TryParseNamedOutputLabel but panics on errors.
func MustParseNamedOutputLabel(target, currentPath string) BuildInput {
	label, err := TryParseNamedOutputLabel(target, currentPath)
	if err != nil {
		panic(err)
	}
	return label
}
