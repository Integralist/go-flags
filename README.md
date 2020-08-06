# go-flags

Abstraction for command-line flag parsing (with no dependencies outside of the Standard Library).

> **Note**: implementation is heavily reliant on the `reflection` package.

## Usage

Create a schema that defines your program flags (along with any commands, and the flags associated with those commands). 

Once you have that schema defiend, then call `flags.Parse()` and pass it a pointer to your schema. 

This will result in the required flags being created while also populating the schema struct with the data provided by the user when running your cli application.

It supports creating both short and long flags (as well as specifying a 'usage' description for each flag) by utilizing golang's 'struct tag' feature.

## Command Line Format

This package expects your CLI program to use the following format:

```
<program> <flags> <command> <command-flags>
```

e.g. `your_app -foo "bar" some_command -baz 123`

## Example

Imagine you want to build a CLI program that has two commands `foo` and `bar`.

Each command has its own set of flags:

- `foo`: `-a/-aaa` (`string`), `-b/-bbb` (`string`).
- `bar`: `-c/-ccc` (`bool`).

But also there are a bunch of top-level, non command specific flags you want to define:

- `-d/-debug`
- `-n/-number`
- `-m/-message`

Here is an example of how a user of your CLI program might call it:

```bash
your_app -debug -n 123 -m "something here" foo -a beepboop -b 666
```

For that example to work, be sure to define the following code within your `main.go`.

```go
package main

import (
	"fmt"
	"os"

	"github.com/integralist/go-flags/flags"
)

type Schema struct {
	Debug   bool   `short:"d" usage:"enable debug level logs"`
	Number  int    `short:"n" usage:"a number field"`
	Message string `short:"m" usage:"a message field"`
	Foo     struct {
		AAA string `short:"a" usage:"does A"`
		BBB string `short:"b" usage:"does B"`
	}
	Bar struct {
		CCC bool `short:"c" usage:"does C"`
	}
}

func main() {
	var s Schema

	err := flags.Parse(&s)
	if err != nil {
		fmt.Printf("error parsing schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nfinal struct: %+v\n", s)
	/*
		Output:

		{
			Debug:true
			Number:123
			Message:something here
			Foo:{
				AAA:beepboop
				BBB:666
			}
			Bar:{
				CCC:false
			}
		}
	*/

	fmt.Printf("\nDebug: %+v\n", s.Debug)   // true
	fmt.Printf("Number: %+v\n", s.Number)   // 123
	fmt.Printf("Message: %+v\n", s.Message) // something here
	fmt.Printf("AAA: %+v\n", s.Foo.AAA)     // beepboop
	fmt.Printf("BBB: %+v\n", s.Foo.BBB)     // 666
	fmt.Printf("CCC: %+v\n", s.Bar.CCC)     // false
}
```
