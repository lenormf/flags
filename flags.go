/*
 * flags.go for flags
 * by lenormf
 */

package flags

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type IntVarOptions struct {
	ShortFlag string
	Required  bool
	NArgs     int

	Default      int
	ValueOnExist int
	Choices      []int
}

type FileVarOptions struct {
	ShortFlag string
	Required  bool
	NArgs     int

	Default      *os.File
	ValueOnExist *os.File
	Mode         string
	Perms        os.FileMode
	// FIXME: implement
	CloseOnExit bool
}

type StringVarOptions struct {
	ShortFlag string
	Required  bool
	NArgs     int

	Default      string
	ValueOnExist string
	Choices      []string
}

type BoolVarOptions struct {
	ShortFlag string
	Required  bool
	NArgs     int

	Default      bool
	ValueOnExist bool
}

type ArgumentParser interface {
	IntVar(interface{}, string, string, *IntVarOptions) error
	FileVar(interface{}, string, string, *FileVarOptions) error
	StringVar(interface{}, string, string, *StringVarOptions) error
	BoolVar(interface{}, string, string, *BoolVarOptions) error

	Parse([]string) ([]string, error)

	PrintHelp()
	CloseAllOpenFiles() error
}

type baseVar struct {
	address interface{}
	flag    string
	help    string
}

type intVar struct {
	baseVar

	options IntVarOptions
}

type fileVar struct {
	baseVar

	options FileVarOptions
}

type stringVar struct {
	baseVar

	options StringVarOptions
}

type boolVar struct {
	baseVar

	options BoolVarOptions
}

type parser struct {
	prog        string
	description string

	vars map[string]interface{}

	open_fds []*os.File
}

var (
	VERSION        = 0x0001
	OnParsingError = DefaultOnParsingErrorCallback
	HelpShortFlag  = "-h"
	HelpLongFlag   = "--help"
)

func find_flag_idx(args []string, flag string) int {
	for i, arg := range args {
		if strings.HasPrefix(arg, flag) {
			return i
		}
	}

	return -1
}

func extract_base_options(addr interface{}, ShortFlag *string, Required *bool, NArgs *int) error {
	// XXX: add new types here
	if v, isIntVarPtr := addr.(*intVar); isIntVarPtr {
		*ShortFlag = v.options.ShortFlag
		*Required = v.options.Required
		*NArgs = v.options.NArgs
	} else if v, isFileVarPtr := addr.(*fileVar); isFileVarPtr {
		*ShortFlag = v.options.ShortFlag
		*Required = v.options.Required
		*NArgs = v.options.NArgs
	} else if v, isStringVarPtr := addr.(*stringVar); isStringVarPtr {
		*ShortFlag = v.options.ShortFlag
		*Required = v.options.Required
		*NArgs = v.options.NArgs
	} else if v, isBoolVarPtr := addr.(*boolVar); isBoolVarPtr {
		*ShortFlag = v.options.ShortFlag
		*Required = v.options.Required
		*NArgs = v.options.NArgs
	} else {
		return fmt.Errorf("Unable to infer the type of the given variable")
	}

	return nil
}

func parse_int_flag(parser ArgumentParser, args []string, idx int, nvar *intVar) (int, error) {
	if nvar.options.NArgs > len(args)-idx {
		OnParsingError(parser, fmt.Errorf("Not enough parameters passed to flag %s (expected %d, got %d)", nvar.baseVar.flag, nvar.options.NArgs, len(args)-idx))
	}

	intPtr, isIntPtr := nvar.baseVar.address.(*int)
	intSlicePtr, isIntSlicePtr := nvar.baseVar.address.(*[]int)

	if !isIntPtr && !isIntSlicePtr {
		return 0, fmt.Errorf("Unable to infer type of the placeholder")
	}

	if isIntPtr && nvar.options.NArgs > 1 {
		OnParsingError(parser, fmt.Errorf("Trying to store multiple values in a single variable (%d parameters set for collection)", nvar.options.NArgs))
	}

	i := 0
	for ; i < nvar.options.NArgs; i++ {
		// FIXME: only 32bit integers are supported, no matter the architecture of the host
		n64, err := strconv.ParseInt(args[idx+i], 0, 32)

		if err != nil {
			OnParsingError(parser, fmt.Errorf("Unable to parse the value given for flag %s: %s", nvar.baseVar.flag, err.Error()))
		}

		n := int(n64)
		if len(nvar.options.Choices) > 0 {
			if idx := sort.SearchInts(nvar.options.Choices, n); idx >= len(nvar.options.Choices) {
				OnParsingError(parser, fmt.Errorf("Invalid value given for flag %s (got %d)", nvar.baseVar.flag, n))
			}
		}

		if isIntSlicePtr {
			*intSlicePtr = append(*intSlicePtr, n)
		} else if isIntPtr {
			*intPtr = n
		}
	}

	return i, nil
}

func parse_file_flag(parser ArgumentParser, args []string, idx int, fvar *fileVar) (int, error) {
	if fvar.options.NArgs > len(args)-idx {
		OnParsingError(parser, fmt.Errorf("Not enough parameters passed to flag %s (expected %d, got %d)", fvar.baseVar.flag, fvar.options.NArgs, len(args)-idx))
	}

	filePtr, isFilePtr := fvar.baseVar.address.(**os.File)
	fileSlicePtr, isFileSlicePtr := fvar.baseVar.address.(*[]*os.File)

	if !isFilePtr && !isFileSlicePtr {
		return 0, fmt.Errorf("Unable to infer type of the placeholder")
	}

	if isFilePtr && fvar.options.NArgs > 1 {
		OnParsingError(parser, fmt.Errorf("Trying to store multiple values in a single variable (%d parameters set for collection)", fvar.options.NArgs))
	}

	i := 0
	for ; i < fvar.options.NArgs; i++ {
		var err error
		var fd *os.File
		arg := args[idx+i]

		switch fvar.options.Mode {
		case "w":
			if fvar.options.Perms > 0 {
				fd, err = os.OpenFile(arg, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fvar.options.Perms)
			} else {
				fd, err = os.OpenFile(arg, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
			}
		case "rw", "wr":
			if fvar.options.Perms > 0 {
				fd, err = os.OpenFile(arg, os.O_CREATE|os.O_TRUNC|os.O_RDWR, fvar.options.Perms)
			} else {
				fd, err = os.OpenFile(arg, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
			}
		case "r":
		default:
			fd, err = os.Open(arg)
		}

		if err != nil {
			OnParsingError(parser, fmt.Errorf("Unable to open file: %s", err))
		} else {
			if isFileSlicePtr {
				*fileSlicePtr = append(*fileSlicePtr, fd)
			} else if isFilePtr {
				*filePtr = fd
			}
		}
	}

	return i, nil
}

func parse_string_flag(parser ArgumentParser, args []string, idx int, svar *stringVar) (int, error) {
	if svar.options.NArgs > len(args)-idx {
		OnParsingError(parser, fmt.Errorf("Not enough parameters passed to flag %s (expected %d, got %d)", svar.baseVar.flag, svar.options.NArgs, len(args)-idx))
	}

	stringPtr, isStringPtr := svar.baseVar.address.(*string)
	stringSlicePtr, isStringSlicePtr := svar.baseVar.address.(*[]string)

	if !isStringPtr && !isStringSlicePtr {
		return 0, fmt.Errorf("Unable to infer type of the placeholder")
	}

	if isStringPtr && svar.options.NArgs > 1 {
		OnParsingError(parser, fmt.Errorf("Trying to store multiple values in a single variable (%d parameters set for collection)", svar.options.NArgs))
	}

	i := 0
	for ; i < svar.options.NArgs; i++ {
		s := args[idx+i]

		if len(svar.options.Choices) > 0 {
			if idx := sort.SearchStrings(svar.options.Choices, s); idx >= len(svar.options.Choices) {
				OnParsingError(parser, fmt.Errorf("Invalid value given for flag %s (got %d)", svar.baseVar.flag, s))
			}
		}

		if isStringSlicePtr {
			*stringSlicePtr = append(*stringSlicePtr, s)
		} else if isStringPtr {
			*stringPtr = s
		}
	}

	return i, nil
}

func parse_bool_flag(parser ArgumentParser, args []string, idx int, bvar *boolVar) (int, error) {
	if bvar.options.NArgs > len(args)-idx {
		OnParsingError(parser, fmt.Errorf("Not enough parameters passed to flag %s (expected %d, got %d)", bvar.baseVar.flag, bvar.options.NArgs, len(args)-idx))
	}

	boolPtr, isBoolPtr := bvar.baseVar.address.(*bool)
	boolSlicePtr, isBoolSlicePtr := bvar.baseVar.address.(*[]bool)

	if !isBoolPtr && !isBoolSlicePtr {
		return 0, fmt.Errorf("Unable to infer type of the placeholder")
	}

	if isBoolPtr {
		if bvar.options.NArgs > 1 {
			OnParsingError(parser, fmt.Errorf("Trying to store multiple values in a single variable (%d parameters set for collection)", bvar.options.NArgs))
		} else if isBoolPtr {
			*boolPtr = bvar.options.ValueOnExist
		}
	}

	i := 0
	for ; i < bvar.options.NArgs; i++ {
		b, err := strconv.ParseBool(strings.ToLower(args[idx+i]))

		if err != nil {
			OnParsingError(parser, fmt.Errorf("Unable to parse the value given for flag %s: %s", bvar.baseVar.flag, err.Error()))
		}

		if isBoolSlicePtr {
			*boolSlicePtr = append(*boolSlicePtr, b)
		} else if isBoolPtr {
			*boolPtr = b
		}
	}

	return i, nil
}

func consume_args(parser ArgumentParser, args []string, idx int, addr interface{}) (int, error) {
	// XXX: add new types here
	if v, isIntVarPtr := addr.(*intVar); isIntVarPtr {
		return parse_int_flag(parser, args, idx+1, v)
	} else if v, isFileVarPtr := addr.(*fileVar); isFileVarPtr {
		return parse_file_flag(parser, args, idx+1, v)
	} else if v, isStringVarPtr := addr.(*stringVar); isStringVarPtr {
		return parse_string_flag(parser, args, idx+1, v)
	} else if v, isBoolVarPtr := addr.(*boolVar); isBoolVarPtr {
		return parse_bool_flag(parser, args, idx+1, v)
	}

	return 0, fmt.Errorf("Unable to infer the type of the given variable")
}

func parse_flags(parser ArgumentParser, vars map[string]interface{}, args []string) ([]string, error) {
	for flag, addr := range vars {
		ShortFlag := ""
		Required := false
		NArgs := 0
		idx := -1

		if !strings.HasPrefix(flag, "-") {
			continue
		}

		if err := extract_base_options(addr, &ShortFlag, &Required, &NArgs); err != nil {
			return args, err
		}

		idx = find_flag_idx(args, flag)
		if idx < 0 {
			if len(ShortFlag) > 0 {
				idx = find_flag_idx(args, ShortFlag)
				if idx < 0 {
					if Required {
						OnParsingError(parser, fmt.Errorf("Missing required flag %s/%s", flag, ShortFlag))
					} else {
						continue
					}
				}
			} else {
				if Required {
					OnParsingError(parser, fmt.Errorf("Missing required flag: %s", flag))
				} else {
					continue
				}
			}
		}

		if eq_idx := strings.Index(args[idx], "="); eq_idx > -1 {
			if eq_idx == len(args[idx])-1 {
				OnParsingError(parser, fmt.Errorf("No value assigned to flag %s", flag))
			}

			param := args[idx][eq_idx+1:]
			args[idx] = args[idx][:eq_idx]

			args = append(args, "")
			copy(args[idx+2:], args[idx+1:])
			args[idx+1] = param
		}

		if nargs, err := consume_args(parser, args, idx, addr); err != nil {
			return args, err
		} else if NArgs > 0 && nargs < NArgs {
			OnParsingError(parser, fmt.Errorf("Not enough parameters passed to flag %s", flag))
		} else {
			var new_args []string

			if idx > 0 {
				new_args = append(new_args, args[0:idx]...)
			}
			if idx+nargs < len(args) {
				new_args = append(new_args, args[nargs+idx+1:]...)
			}

			args = new_args
		}
	}

	return args, nil
}

func parse_positionals(parser ArgumentParser, vars map[string]interface{}, args []string) ([]string, error) {
	max_length_collected := 0

	for flag, addr := range vars {
		Required := false
		NArgs := 0

		if strings.HasPrefix(flag, "-") {
			continue
		}

		if err := extract_base_options(addr, new(string), &Required, &NArgs); err != nil {
			return args, err
		}

		svar, isStringVarPtr := addr.(*stringVar)
		if !isStringVarPtr {
			return nil, fmt.Errorf("Unable to infer type of the internal variable for flag %s", flag)
		}

		stringPtr, isStringPtr := svar.baseVar.address.(*string)
		stringSlicePtr, isStringSlicePtr := svar.baseVar.address.(*[]string)

		if !isStringPtr && !isStringSlicePtr {
			return nil, fmt.Errorf("Unable to infer type of the placeholder for flag %s", flag)
		}

		length_collected := 0
		if NArgs > 0 {
			if len(args) < NArgs && Required {
				OnParsingError(parser, fmt.Errorf("Not enough arguments passed to positional flag %s for collection (expected %d, got %d)", flag, NArgs, len(args)))
			} else if len(args) > 0 {
				if isStringSlicePtr {
					length_collected = int(math.Min(float64(len(args)), float64(NArgs)))
					*stringSlicePtr = append(*stringSlicePtr, args[:length_collected]...)
				} else if isStringPtr {
					length_collected = 1
					*stringPtr = args[0]
				}
			}
		} else {
			if len(args) == 0 && Required {
				OnParsingError(parser, fmt.Errorf("No arguments passed to flag %s for collection", flag))
			} else if len(args) > 0 {
				if isStringSlicePtr {
					length_collected = len(args)
					*stringSlicePtr = append(*stringSlicePtr, args[:len(args)]...)
				} else if isStringPtr {
					length_collected = 1
					*stringPtr = args[0]
				}
			}
		}

		if length_collected > max_length_collected {
			max_length_collected = length_collected
		}
	}

	args = args[max_length_collected:]

	return args, nil
}

func NewArgumentsParser(prog, description string) ArgumentParser {
	return &parser{
		prog:        prog,
		description: description,
		vars:        make(map[string]interface{}),
	}
}

func DefaultOnParsingErrorCallback(parser ArgumentParser, err error) {
	fmt.Printf("%s\n", err.Error())
	parser.PrintHelp()
	os.Exit(1)
}

func (this *parser) IntVar(address interface{}, flag string, help string, options *IntVarOptions) error {
	if _, ok := this.vars[flag]; ok == true {
		return fmt.Errorf("Flag \"%s\" was already added to the parser", flag)
	}

	if options.NArgs == 0 {
		options.NArgs = 1
	}

	this.vars[flag] = &intVar{
		baseVar: baseVar{
			address: address,
			flag:    flag,
			help:    help,
		},
		options: *options,
	}

	return nil
}

func (this *parser) FileVar(address interface{}, flag string, help string, options *FileVarOptions) error {
	if _, ok := this.vars[flag]; ok == true {
		return fmt.Errorf("Flag \"%s\" was already added to the parser", flag)
	}

	if options.NArgs == 0 {
		options.NArgs = 1
	}

	this.vars[flag] = &fileVar{
		baseVar: baseVar{
			address: address,
			flag:    flag,
			help:    help,
		},
		options: *options,
	}

	if options.CloseOnExit {
		if fd, isFilePtr := address.(**os.File); isFilePtr {
			this.open_fds = append(this.open_fds, *fd)
		} else if fds, isFileSlicePtr := address.(*[]*os.File); isFileSlicePtr {
			this.open_fds = append(this.open_fds, *fds...)
		} else {
			return fmt.Errorf("Invalid address type passed")
		}
	}

	return nil
}

func (this *parser) StringVar(address interface{}, flag string, help string, options *StringVarOptions) error {
	if _, ok := this.vars[flag]; ok == true {
		return fmt.Errorf("Flag \"%s\" was already added to the parser", flag)
	}

	this.vars[flag] = &stringVar{
		baseVar: baseVar{
			address: address,
			flag:    flag,
			help:    help,
		},
		options: *options,
	}

	return nil
}

func (this *parser) BoolVar(address interface{}, flag string, help string, options *BoolVarOptions) error {
	if _, ok := this.vars[flag]; ok == true {
		return fmt.Errorf("Flag \"%s\" was already added to the parser", flag)
	}

	this.vars[flag] = &boolVar{
		baseVar: baseVar{
			address: address,
			flag:    flag,
			help:    help,
		},
		options: *options,
	}

	return nil
}

func (this *parser) Parse(args []string) ([]string, error) {
	unparsed_args, err := parse_flags(this, this.vars, args)
	if err != nil {
		return nil, err
	}

	// We check for the -h/--help flags after processing the arguments in order
	// not to trigger a false positive if those strings are passed as flag
	// arguments
	// TODO: implement --
	if idx := math.Min(float64(sort.SearchStrings(unparsed_args, HelpShortFlag)), float64(sort.SearchStrings(unparsed_args, HelpLongFlag))); int(idx) < len(args) {
		this.PrintHelp()
		os.Exit(0)
	}

	return parse_positionals(this, this.vars, unparsed_args)
}

func (this *parser) PrintHelp() {
	// FIXME: implement
	fmt.Printf("%s - %s\n", this.prog, this.description)
}

func (this *parser) CloseAllOpenFiles() error {
	for i, fd := range this.open_fds {
		if err := fd.Close(); err != nil {
			this.open_fds = this.open_fds[i:]
			return err
		}
	}

	this.open_fds = []*os.File{}

	return nil
}
