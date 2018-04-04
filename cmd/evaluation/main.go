package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/velovix/lambda-starship-user-stats/datatypes"
)

// event represents an event of some kind in the game.
type event interface {
	fmt.Stringer
	getTimestamp() int64
	value() string
}

// errorEvent is an event as the result of an error.
type errorEvent datatypes.ErrorInstance

func (e errorEvent) getTimestamp() int64 {
	return e.Timestamp
}

func (e errorEvent) value() string {
	return e.Description
}

func (e errorEvent) String() string {
	return "Error: " + e.Description
}

// replEvent is an event representing a run command in the REPL.
type replEvent datatypes.REPLCommand

func (r replEvent) getTimestamp() int64 {
	return r.Timestamp
}

func (r replEvent) value() string {
	return r.Command
}

func (r replEvent) String() string {
	return "REPL : " + r.Command
}

type editorEvent datatypes.EditorContent

func (e editorEvent) getTimestamp() int64 {
	return e.Timestamp
}

func (e editorEvent) value() string {
	return e.Content
}

func (e editorEvent) String() string {
	out := "Editor:\n"

	for _, line := range strings.Split(e.Content, "\n") {
		out += "    " + line + "\n"
	}

	return out
}

type session struct {
	uid    string
	events []event
}

// newSession creates a new session from the given UID containing all its
// events.
func newSession(ctx context.Context, client *datastore.Client, uid string) (session, error) {
	sess := session{uid: uid}

	// Get all errors
	query := datastore.NewQuery(datatypes.ErrorInstanceKind).
		Filter("UID =", uid)
	var errorInstances []datatypes.ErrorInstance
	if _, err := client.GetAll(ctx, query, &errorInstances); err != nil {
		return session{}, err
	}

	for _, instance := range errorInstances {
		sess.events = append(sess.events, errorEvent(instance))
	}

	// Get all REPL commands
	query = datastore.NewQuery(datatypes.REPLCommandKind).
		Filter("UID =", uid)
	var replCommands []datatypes.REPLCommand
	if _, err := client.GetAll(ctx, query, &replCommands); err != nil {
		return session{}, err
	}

	for _, cmd := range replCommands {
		sess.events = append(sess.events, replEvent(cmd))
	}

	// Get all editor saves
	query = datastore.NewQuery(datatypes.EditorContentKind).
		Filter("UID =", uid)
	var editorContents []datatypes.EditorContent
	if _, err := client.GetAll(ctx, query, &editorContents); err != nil {
		return session{}, err
	}

	for _, editorContent := range editorContents {
		sess.events = append(sess.events, editorEvent(editorContent))
	}

	sort.Slice(sess.events, func(i, j int) bool {
		return sess.events[i].getTimestamp() < sess.events[j].getTimestamp()
	})

	return sess, nil
}

type commandAndError struct {
	cmd datatypes.REPLCommand
	err *datatypes.ErrorInstance
}

func (u *session) commandAndErrors() []commandAndError {
	var output []commandAndError

	var lastCmd *datatypes.REPLCommand

	for _, event := range u.events {
		if cmd, ok := event.(replEvent); ok {
			if lastCmd != nil {
				output = append(output, commandAndError{
					*lastCmd,
					nil})
			}

			replCommand := datatypes.REPLCommand(cmd)
			lastCmd = &replCommand
		} else if err, ok := event.(errorEvent); ok {
			errorInstance := datatypes.ErrorInstance(err)
			output = append(output, commandAndError{
				*lastCmd,
				&errorInstance})
			lastCmd = nil
		}
	}

	return output
}

// errPatterns is a map whose keys are small descriptions of an error type and
// values are regular expressions that match on errors of that type.
var errPatterns = map[string]*regexp.Regexp{
	"UnknownCallable":      regexp.MustCompile("Unknown callable '(.*)'"),
	"VariableHasNoValue":   regexp.MustCompile("Variable ([^\\s]+) has no value"),
	"InvalidNumberOfArgs":  regexp.MustCompile("Invalid number of args"),
	"CallableMustBeSymbol": regexp.MustCompile("Callable name must be a symbol"),
	"NoSwitchWithID":       regexp.MustCompile("No such switch with ID ([^\\s]+) exists"),
	"PropellantGenerator":  regexp.MustCompile("Propellant cannot be powered with backup generator"),
	"LightGenerator":       regexp.MustCompile("Light cannot be powered with backup generator"),
	"NoThrusterWithID":     regexp.MustCompile("No thruster with ID ([^\\s]+) exists"),
	"ArugmentMustBeOfType": regexp.MustCompile("Argument ([^\\s]+) must be of type ([^\\s]+), got ([^\\s]+)"),
	"TooManyArguments":     regexp.MustCompile("Too many arguments"),
	"ArgsMustBeNumbers":    regexp.MustCompile("All arguments to (.) must be numbers"),
}

// errorTypeCount returns the count of all errors in the database, segregated
// by their "type", as mandated by errPatterns.
func errorTypeCount(ctx context.Context, client *datastore.Client) (map[string]int, error) {
	query := datastore.NewQuery(datatypes.ErrorInstanceKind)

	var errorInstances []datatypes.ErrorInstance
	if _, err := client.GetAll(ctx, query, &errorInstances); err != nil {
		return nil, fmt.Errorf("getting instances: %v", err)
	}

	log.Println("Got", len(errorInstances), "error instances")

	matchCnt := make(map[string]int)

	for _, errorInstance := range errorInstances {
		for name, pattern := range errPatterns {
			if pattern.MatchString(errorInstance.Description) {
				matchCnt[name]++
				break
			}
		}
	}

	return matchCnt, nil
}

type variableHasNoValueInfo struct {
	variable string
	count    int
}

// variableHasNoValueCount finds how many instances of each variable name
// resulted in a "VariableHasNoValue" error.
func variableHasNoValueCount(ctx context.Context, client *datastore.Client) ([]variableHasNoValueInfo, error) {
	query := datastore.NewQuery(datatypes.ErrorInstanceKind)

	var errorInstances []datatypes.ErrorInstance
	if _, err := client.GetAll(ctx, query, &errorInstances); err != nil {
		return nil, fmt.Errorf("getting instances: %v", err)
	}

	instanceCnt := make(map[string]int)

	for _, errorInstance := range errorInstances {
		variable := errPatterns["VariableHasNoValue"].FindString(errorInstance.Description)
		if variable != "" {
			instanceCnt[variable]++
		}
	}

	var sorted []variableHasNoValueInfo
	for variable, cnt := range instanceCnt {
		sorted = append(sorted, variableHasNoValueInfo{
			variable: variable,
			count:    cnt})
	}

	sort.Slice(sorted, func(i, j int) bool {
		// Reverse the sort
		return sorted[i].count > sorted[j].count
	})

	return sorted, nil
}

// editorUse returns the quantity of sessions that used the editor.
func editorUse(ctx context.Context, client *datastore.Client, uids []string) (int, error) {
	usedEditorCount := 0

	for _, uid := range uids {
		query := datastore.NewQuery(datatypes.EditorContentKind).
			Filter("UID =", uid)

		count, err := client.Count(ctx, query)
		if err != nil {
			return 0, err
		}

		if count > 0 {
			usedEditorCount++
		}
	}

	return usedEditorCount, nil
}

// getUIDs returns all unique UIDs in the database.
func getUIDs(ctx context.Context, client *datastore.Client) ([]string, error) {
	query := datastore.NewQuery(datatypes.REPLCommandKind)

	var replCommands []datatypes.REPLCommand
	if _, err := client.GetAll(ctx, query, &replCommands); err != nil {
		return nil, fmt.Errorf("getting REPL instances: %v", err)
	}

	// Use map keys as a ramshackle "set" type
	set := make(map[string]struct{})
	for _, cmd := range replCommands {
		set[cmd.UID] = struct{}{}
	}

	var output []string
	for val := range set {
		output = append(output, val)
	}

	return output, nil
}

func main() {
	ctx := context.Background()

	client, err := datastore.NewClient(ctx, "lambda-starship-user-stats")
	if err != nil {
		log.Fatalf("creating Datastore client: %v", err)
	}

	matchCnt, err := errorTypeCount(ctx, client)
	if err != nil {
		panic(err)
	}
	log.Println("--- Error Frequency ---")
	for name, cnt := range matchCnt {
		log.Printf("%v: %v", name, cnt)
	}

	file, err := os.Create("user-sessions.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Get the variable frequency of VariableHasNoValue errors
	varsWithNoValue, err := variableHasNoValueCount(ctx, client)
	if err != nil {
		panic(err)
	}
	log.Println("--- VariableHasNoValue top variables ---")
	for _, varWithNoValue := range varsWithNoValue {
		log.Printf("%v: %v", varWithNoValue.variable, varWithNoValue.count)
	}

	uids, err := getUIDs(ctx, client)
	if err != nil {
		panic(err)
	}

	editorUseCount, err := editorUse(ctx, client, uids)
	if err != nil {
		panic(err)
	}
	log.Printf("%v sessions used the editor out of %v", editorUseCount, len(uids))

	// Get the errors, commands, and editor saves from each user session
	for _, uid := range uids {
		sess, err := newSession(ctx, client, uid)
		if err != nil {
			panic(err)
		}

		for _, e := range sess.events {
			file.WriteString(e.String() + "\n")
		}
	}
	log.Printf("Wrote session info to user-sessions.txt")
}
