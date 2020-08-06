package flags

import (
	"errors"
	"flag"
	"os"
	"reflect"
	"strings"
)

// cmds is used for tracking the 'commands' defined in the user provided struct
//
// TODO: avoid package level scoped variables by injecting as a dependency.
//
var cmds = make(map[string]bool)

var (
	ErrNoArgs    = errors.New("no flags or commands provided")
	ErrWrongType = errors.New("expected a pointer to a struct for the schema")
)

func Parse(s interface{}) error {
	args := os.Args[1:]
	if len(args) == 0 {
		return ErrNoArgs
	}

	// ValueOf() returns the concrete struct value (e.g. &{...})
	// Indirect() returns the value that is pointed to (e.g. the actual struct)
	//
	v := reflect.Indirect(reflect.ValueOf(s))

	// we acquire the type of the value (e.g. main.Schema)
	//
	// NOTE: we could have done this like so reflect.TypeOf(s).Elem() but I find
	// calling Type() on the actual value looks a bit cleaner :shrugs:
	//
	st := v.Type()

	// we code defensively and ensure a struct was provided, otherwise we'll have
	// to raise an error to avoid panics later on in the code where we're
	// presuming a struct was given.
	//
	if v.Kind() != reflect.Struct {
		return ErrWrongType
	}

	// TODO: redesign the IterFields function.
	//
	// it works but it's fugly as hell.
	// having to have a control var like `recurse` is nasty.
	//
	recurse := false

	// iterate over the top level fields of the user provided struct,
	// and create the required flags.
	//
	IterFields(recurse, st, v, func(field reflect.Value, sf reflect.StructField, cmd ...string) {
		switch field.Kind() {
		case reflect.Bool:
			var v bool
			flag.BoolVar(&v, strings.ToLower(sf.Name), false, sf.Tag.Get("usage"))
			flag.BoolVar(&v, sf.Tag.Get("short"), false, sf.Tag.Get("usage")+" (shorthand)")
		case reflect.Int:
			var v int
			flag.IntVar(&v, strings.ToLower(sf.Name), 0, sf.Tag.Get("usage"))
			flag.IntVar(&v, sf.Tag.Get("short"), 0, sf.Tag.Get("usage")+" (shorthand)")
		case reflect.String:
			var v string
			flag.StringVar(&v, strings.ToLower(sf.Name), "", sf.Tag.Get("usage"))
			flag.StringVar(&v, sf.Tag.Get("short"), "", sf.Tag.Get("usage")+" (shorthand)")
		}
	})

	flag.Parse()

	// iterate over the top level fields of the user provided struct,
	// and populate the fields with the parsed flag values.
	//
	IterFields(recurse, st, v, func(field reflect.Value, sf reflect.StructField, cmd ...string) {
		flag.Visit(func(f *flag.Flag) {
			// annoyingly you can't get to the flag's concrete value, so we have to
			// first type assert it to a flag.Getter which then gives us an interface
			// (e.g. Get()) for accessing the internal value which we finally can
			// type assert into the correct value type (and thus we can assign that
			// to our struct field).
			//
			getter, ok := f.Value.(flag.Getter)
			if ok {
				if f.Name == strings.ToLower(sf.Name) || f.Name == sf.Tag.Get("short") {
					switch field.Kind() {
					case reflect.Bool:
						if b, ok := getter.Get().(bool); ok {
							field.Set(reflect.ValueOf(b))
						}
					case reflect.Int:
						if i, ok := getter.Get().(int); ok {
							field.Set(reflect.ValueOf(i))
						}
					case reflect.String:
						if s, ok := getter.Get().(string); ok {
							field.Set(reflect.ValueOf(s))
						}
					}
				}
			}
		})
	})

	cmd := IdentifyCommand(cmds, args)
	cmdFlags := CommandFlags(cmd, flag.Args())

	cfs := CommandFlagSet(cmd, cmdFlags, st, v)
	err := cfs.Parse(cmdFlags)
	if err != nil {
		return err
	}

	recurse = true

	// iterate over the command fields of the user provided struct,
	// and populate the fields with the parsed flagset values.
	//
	IterFields(recurse, st, v, func(field reflect.Value, sf reflect.StructField, cmd ...string) {
		cfs.Visit(func(f *flag.Flag) {
			// annoyingly you can't get to the flag's concrete value, so we have to
			// first type assert it to a flag.Getter which then gives us an interface
			// (e.g. Get()) for accessing the internal value which we finally can
			// type assert into the correct value type (and thus we can assign that
			// to our struct field).
			//
			getter, ok := f.Value.(flag.Getter)
			if ok {
				if f.Name == strings.ToLower(sf.Name) || f.Name == sf.Tag.Get("short") {
					switch field.Kind() {
					case reflect.Bool:
						if b, ok := getter.Get().(bool); ok {
							field.Set(reflect.ValueOf(b))
						}
					case reflect.Int:
						if i, ok := getter.Get().(int); ok {
							field.Set(reflect.ValueOf(i))
						}
					case reflect.String:
						if s, ok := getter.Get().(string); ok {
							field.Set(reflect.ValueOf(s))
						}
					}
				}
			}
		})
	})

	return nil
}

// IterFields iterates over all fields of a struct, including nested structs,
// and processes their individual fields by passing them into a callback.
//
func IterFields(recurse bool, st reflect.Type, v reflect.Value, callback func(f reflect.Value, sf reflect.StructField, cmd ...string)) {
	// NOTE: if we're passed something that isn't a struct, then the program will
	// panic when we call NumField() as this is the reality of using reflection.
	//
	// we are relying on the consumer of this package to follow the instructions
	// given and to provide us with what we are expecting.
	//
	// so if we're not careful, then we violate the language type safety.
	// but we protect against this in the calling function by checking for a
	// struct before calling IterFields.
	//
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// we call Field() on the struct type so we can get a StructField type,
		// which we have to do in order to access the struct 'tags' on the field.
		//
		// it also gives us access to the field name so we can create the various
		// flags necessary (as well as determine the command that a user runs).
		//
		sf := st.Field(i)

		if field.Kind() == reflect.Struct {
			// when we see a struct we expect by convention for this to be a
			// 'command' that will have its own set of flags.
			//
			cmd := strings.ToLower(sf.Name)
			if _, ok := cmds[cmd]; !ok {
				cmds[cmd] = true
			}

			// we use CanInterface() because otherise if we were to call Interface()
			// on a field that was unexported, then the program would panic.
			//
			if recurse && field.CanInterface() {
				// we use Interface() to get the nested struct value as an interface{}.
				// this is done because if we called TypeOf on the field variable, then
				// we would end up with reflect.Value when really we need the nested
				// struct's concrete type definition (e.g. struct {...}).
				//
				st := reflect.TypeOf(field.Interface())

				for i := 0; i < field.NumField(); i++ {
					// again, we get the field from the nested struct, as well as acquire
					// its StructField type for purposes already explained above.
					//
					field := field.Field(i)
					st := st.Field(i)

					// because our callback function is going to attempt to set values on
					// these struct fields, we need to be sure they are 'settable' first.
					//
					if field.CanSet() {
						callback(field, st, cmd)
					}
				}
			}
		} else {
			// we check if recurse is false because we don't want our nested commands
			// to accidentally add the top-level fields into our command flagset and
			// thus -h/--help would show the top-level fields in the help output.
			//
			// also, because our callback function is going to attempt to set values
			// on these struct fields, we need to be sure they are 'settable' first.
			//
			//
			if !recurse && field.CanSet() {
				callback(field, sf)
			}
		}
	}
}

// IdentifyCommand parses the arguments provided looking for a 'command'.
//
// this implementation presumes that the format of the arguments will be...
//
// <program> <flag(s)> <command> <flag(s) for command>
//
func IdentifyCommand(cmds map[string]bool, args []string) string {
	commandIndex := 0
	commandSeen := false

	for _, arg := range args {
		if commandSeen {
			break
		}

		if strings.HasPrefix(arg, "-") == true {
			commandIndex++
			continue
		}

		for cmd := range cmds {
			if arg == cmd {
				commandSeen = true
				break
			}
		}

		if !commandSeen {
			commandIndex++
		}
	}

	if !commandSeen {
		return ""
	}

	return args[commandIndex]
}

// CommandFlags parses the flags that are provided after the 'command'.
//
func CommandFlags(cmd string, args []string) []string {
	for i, v := range args {
		if v == cmd {
			return args[i+1:]
		}
	}

	return []string{}
}

// CommandFlagSet defines flags for the command as a FlagSet.
//
func CommandFlagSet(cmd string, cmdFlags []string, st reflect.Type, v reflect.Value) *flag.FlagSet {
	cfs := flag.NewFlagSet(cmd, flag.ExitOnError)
	recurse := true

	// iterate over the nested fields of the user provided struct,
	// and create the required flagset flags.
	//
	IterFields(recurse, st, v, func(field reflect.Value, sf reflect.StructField, currentCmd ...string) {
		// we're overloading the use of variadic functions to allow some iterations
		// over our struct to pass a cmd, and others that aren't a command to not.
		//
		// this means when we explicitly access the first index, there isn't ever
		// any expectation for there to be more than one command passed through.
		//
		if currentCmd[0] == cmd {
			switch field.Kind() {
			case reflect.Bool:
				var v bool
				cfs.BoolVar(&v, strings.ToLower(sf.Name), false, sf.Tag.Get("usage"))
				cfs.BoolVar(&v, sf.Tag.Get("short"), false, sf.Tag.Get("usage")+" (shorthand)")
			case reflect.Int:
				var v int
				cfs.IntVar(&v, strings.ToLower(sf.Name), 0, sf.Tag.Get("usage"))
				cfs.IntVar(&v, sf.Tag.Get("short"), 0, sf.Tag.Get("usage")+" (shorthand)")
			case reflect.String:
				var v string
				cfs.StringVar(&v, strings.ToLower(sf.Name), "", sf.Tag.Get("usage"))
				cfs.StringVar(&v, sf.Tag.Get("short"), "", sf.Tag.Get("usage")+" (shorthand)")
			}
		}
	})

	return cfs
}
