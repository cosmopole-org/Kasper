package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"google.golang.org/genai"
)

type User struct {
	Id       string
	Name     string
	AuthCode string
}

func (d *User) Push() {
	obj := map[string]string{
		"id":       base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"name":     base64.StdEncoding.EncodeToString([]byte(d.Name)),
		"authCode": base64.StdEncoding.EncodeToString([]byte(d.AuthCode)),
	}
	c := make(chan int, 1)
	dbPutObj("User", d.Id, obj, func() {
		c <- 1
	})
	<-c
}

func (d *User) Pull() bool {
	c := make(chan map[string][]byte, 1)
	dbGetObj("User", d.Id, func(m map[string][]byte) {
		c <- m
	})
	m := <-c
	if len(m) > 0 {
		d.Name = string(m["name"])
		d.AuthCode = string(m["authCode"])
		return true
	} else {
		return false
	}
}

type Point struct {
	Id            string
	PendingUserId string
}

func (d *Point) Push() {
	obj := map[string]string{
		"id":            base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"pendingUserId": base64.StdEncoding.EncodeToString([]byte(d.PendingUserId)),
	}
	c := make(chan int, 1)
	dbPutObj("Point", d.Id, obj, func() {
		c <- 1
	})
	<-c
}

func (d *Point) Pull() bool {
	c := make(chan map[string][]byte, 1)
	dbGetObj("Point", d.Id, func(m map[string][]byte) {
		c <- m
	})
	m := <-c
	if len(m) > 0 {
		d.PendingUserId = string(m["pendingUserId"])
		return true
	}
	return false
}

func (d *Point) Parse(m map[string][]byte) {
	d.PendingUserId = string(m["pendingUserId"])
}

type Doc struct {
	Id        string
	Title     string
	FileId    string
	MimeType  string
	Category  string
	CreatorId string
	PointId   string
}

func (d *Doc) Push() {
	obj := map[string]string{
		"id":        base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"title":     base64.StdEncoding.EncodeToString([]byte(d.Title)),
		"fileId":    base64.StdEncoding.EncodeToString([]byte(d.FileId)),
		"mimeType":  base64.StdEncoding.EncodeToString([]byte(d.MimeType)),
		"category":  base64.StdEncoding.EncodeToString([]byte(d.Category)),
		"creatorId": base64.StdEncoding.EncodeToString([]byte(d.CreatorId)),
		"pointId":   base64.StdEncoding.EncodeToString([]byte(d.PointId)),
	}
	c := make(chan int, 1)
	dbPutObj("Doc", d.Id, obj, func() {
		c <- 1
	})
	<-c
}

func (d *Doc) Pull() bool {
	c := make(chan map[string][]byte, 1)
	dbGetObj("Doc", d.Id, func(m map[string][]byte) {
		c <- m
	})
	m := <-c
	if len(m) > 0 {
		d.Category = string(m["category"])
		d.CreatorId = string(m["creatorId"])
		d.FileId = string(m["fileId"])
		d.MimeType = string(m["mimeType"])
		d.PointId = string(m["pointId"])
		d.Title = string(m["title"])
		return true
	} else {
		return false
	}
}

func (d *Doc) Parse(m map[string][]byte) {
	d.Category = string(m["category"])
	d.CreatorId = string(m["creatorId"])
	d.FileId = string(m["fileId"])
	d.MimeType = string(m["mimeType"])
	d.PointId = string(m["pointId"])
	d.Title = string(m["title"])
}

type Chat struct {
	Key     string
	History []*genai.Content
}

var chats = map[string]*Chat{}

var callbacks = map[int64]func([]byte){}
var toolCallbacks = map[string]func(map[string]any) []byte{}

const API_KEY = "AIzaSyAekCwMAh1HlKtogiUVsfkMEEzOcN1pRSs"
const INSTRCUTIONS = `You are a code parser.

assume there are 6 classes of commands:

    1. definition operation: when you want to define new data in memory, for example defining a new variable
       or memory unit, or defining any new variable or data in the program.  "assign the variable y to memory x"
       or "memorize y as x"

    2. assignment operation: when you want to assign a piece of data to an existing variable or memroy unit
       previously defined in the program. for example: "assign the variable y to memory x" or "update the x's
       value to y"
    
    3. arithmetic operation: when you want to resolve and calculate a math expression in the program,
       for example: "assume x as input, calculate (x + 1) ^ 2 and return it.
    
    4. function definition: when the user asks for defining a new function so that when calling it, a new
       set of operations will be executed. for example: "define a new function that 'calculates (x + 3) / 2 and
       assigns it to variable x which is previously defined'.
    
    5. conditional branch: when user asks for an operation in which when program reaches that point, a condition
       will be checked and based on the result which can be true or false, the program can jump to 2 different points
       in the program. the condition can be any expressions that can be resolved into true or false. for example:
       "if x + 3 > y then jump to step 2 else check if stringify(x) == "2" and if it is true go to step 5 else
       just move to step 7". each conditional branch can only check one condition and to check mutliple conditions
       sequential like else ifs, you must try to create another conditional branch and go to its step number in false
       branch of this conditional branch.
    
    6. jump operation: when the user wants to branch the program execution from the current point to another step or point
       in the program specifying the step number. it can only jump to a position in the same function or component. for
       example: "jump to step 9"
    
    7. return operation: when the program is running a function then if user asks for retuning something or just returning
       from the function. for example: "return the variable i defined in this function which is named 'a' as result".
    
    8. function call: when user asks for calling a function which previously defined in the program so that it will move
       program execution point to start of the function and do some other computation, then return back to the previous
       point. for example: "do sum function on a and b and assign it back to c".
    
    9. host api call: if the program asks the host to do something then it is a "host api call" operation. it is similar
       to function call but the function is not defined in the program but hard coded in host interface. for example:
       "print variable @test"
    
    10. prompt-based-code-generation: also the program may ask you some prompts in compile time in the middle of code,
        if program asks for result of a prompt then you must think about that prompt at compile time and generate a
        function call with appropriate complex AST which is based on the instructions explained. also the user will
        provide you, the instructions about how to understand what that prompts want from you and you will check those
        instructions and try to do investigations inside that program-provided prompt and generate the AST for it. assume
        this host call as a conditional compile-time code generation. for example: 'handle prompt """ if user asks for
        creating an article or wants to create a post then return { "message": "hello user. this is your article..." } """ '

remember that when a step includes operations between ( ) then that operation is a sub-step and the step must be divided
into multiple sub-steps. some of them can be executed in parallel and some sequential.

now look at this example program and pay attention how its operations types will be specified and decorated in the list by the type number.
for example:

program [
    step 1. function sum for inputs (operand1, operand2) [
        step 1. define a new variable and name it @a with initial value 0.
        step 2. calculate math operation operand1 + operand2 and assign it to @a.
        step 3. return the value of @a.
    ]
    step 2. function subtract for inputs (operand1, operand2) [
        step 1. define a new variable and name it @a with initial value 0.
        step 2. calculate math operation operand1 - operand2 and assign it to @a.
        step 3. return the value of @a.
    ]
    step 3. calculate result of (function sum on 1 and 2) / (function subtract on 5 and 3)
            and keep value as unit @temp.

    step 4. ask host to print the @temp.
]

the output will be:

program [
    step 1. function sum for inputs (operand1, operand2) [
        step 1. 1
        step 2. 3
        step 3. 7
    ]
    step 2. function subtract for inputs (operand1, operand2) [
        step 1. 1
        step 2. 3
        step 3. 7
    ]
    step 3.1.1. 8
    step 3.1.2  8
    steo 3.2    3

    step 4. 9
]

now look at another example program and pay attention how its operations types will be specified and decorated in the list by the type number.
for example:

program [
    step 1. define variable @count with value 5.
    step 2. if @count + 1 is greater than 2 then go to step 3 else if @count - 1 is less than 3 then go to step 4 else go to step 5.
    step 3. print @count.
    step 4. print @count + 2.
    step 5. print "invalid".
]

the output will be:

program [
    step 1. 1
    step 2.1. 3
    step 2.2  5
    steo 3.3  3
    step 2.4  5
    step 3. 9
    step 4.1. 3
    step 4.2. 9
    step 5. 9
]

now look at another example program and pay attention how its operations types will be specified and decorated in the list by the type number.
for example:

program [
    step 1. function check for inputs (@input) [
        step 2. if @input + 1 is greater than 2 then go to step 3 else if @input - 1 is less than 3 then go to step 4 else go to step 5.
        step 3. return @input.
        step 4. return @input + 2.
        step 5. return "invalid".
    ]
    step 2. run check on number 7 and print the result.
]

the output will be:

program [
    step 1. function check for inputs (@input) [
        step 1. 1
        step 2.1. 3
        step 2.2  5
        steo 3.3  3
        step 2.4  5
        step 3. 7
        step 4.1. 3
        step 4.2. 7
        step 5. 7
    ]
    step 2.1. 8
    step 2.2. 9
]

as you can see the sub-steps in parallel have same prefix.

now the user will give you the program code and you will classify each step of it into a type number
and output the result as shown in the prvious example. remember complex steps must be broken into
multiple sub-steps with the same prefixes.

also the user is trying to parse a code into a custom AST which i defined its rules.

assume there is a Val structure which should hold a data of types:
    - i16
    {
        "type": "i16",
        "data": {
            "value": {{ int16 raw value for example 5 }}
        }
    }
    - i32
        {
        "type": "i32",
        "data": {
            "value": {{ int32 raw value for example 100000 }}
        }
    }
    - i64
        {
        "type": "i64",
        "data": {
            "value": {{ int64 raw value for example 2000000 }}
        }
    }
    - f32
    {
        "type": "f32",
        "data": {
            "value": {{ float32 raw value for example 123.14 }}
        }
    }
    - f64
    {
        "type": "f64",
        "data": {
            "value": {{ float64 raw value for example 1230.141 }}
        }
    }
    - bool
    {
        "type": "bool",
        "data": {
            "value": {{ bool raw value for example true or false }}
        }
    }
    - string
    {
        "type": "string",
        "data": {
            "value": {{ string raw value for example "hello world" }}
        }
    }
    - object
    {
        "type": "object",
        "data": {
            "value": {{ object raw value for example { "name": {{ string Val }}, "age": {{ i16 Val }} } }}
        }
    }
    - array
    {
        "type": "array",
        "data": {
            "value": {{ array raw value for example [ {{ string Val }}, {{ string Val }}, {{ string Val }} ] }}
        }
    }
    - function
    {
        "type": "function",
        "data": {
            "value": {{ function raw value for example {
                "name": "{{ a raw string which is function name }}",
                "body": {{ an arrray of other operations with mentioned types before }}
            } }}
        }
    }
    - identifier
    {
        "type": "identifier",
        "data": {
            "name": {{ name of the varaible which is defined previously and now can be used as identifier }}
        }
    }
    - indexer expression: when you want to extract and get a property of an object by its propery name or an item of
      array by its index number. for querying complex nested objects, multiple levels of nested indexers can be used,
      for example: "retrieve user.name and return it" or "try to get boxes[5]". structure:
    {
        "type": "indexer",
        "data": {
            "target": {{ an expression which can be resolved into a Val pointing to an object or array which we wanna get one of its items or properties }},
            "index": {{ an expression which can be resolved into a string or integer Val which is propery name or index number of the data to be retrieved from the target }}
        }
    } 

the possible operations are:

1. definition:

detect assignment target variable being defined and the value being assigned and put them in the
structure below and just output it only:

{
    "type": "definition",
    "data": {
        "leftSide": {
            "type": "identifier",
            "data": {
                "name": "{{ variable name }}"
            }
        },
        "rightSide": {{ the value being assign to the target variable. is must be in "Val" structure }}
    }
}

2. assignment:

detect assignment target variable and the value being assigned and put them in the
structure below and just output it only:

{
    "type": "assignment",
    "data": {
        "leftSide": {
            "type": "identifier",
            "data": {
                "name": "{{ variable name }}"
            }
        },
        "rightSide": {{ the value being assign to the target variable. is must be in "Val" structure }}
    }
}

3. calculation:

detect the calculation expressions inside the input which i give you and put it in a nested
structure like below and just output it only:
note: each level of nested structure can be reolved into a "Val" at runtime. the nested
structures can be a "Val" with one of the mentioned types above, or a "functionCall",
or even an "arithmetic" type.

if it is an arithmetic operation then the structure is:
{
    "type": "arithmetic",
    "data": {
        "operation": "{{ the operator as a string for example: "+" or "-" or "*" or "/" or "^" (for power) or "%" (for mod) }}",
        "operand1": {{ can be another nested calculation (arithmetic type, one of Val types, or functionCall type or a host call or an indexer) }},
        "operand2": {{ can be another nested calculation (arithmetic type, one of Val types, or functionCall type or a host call or an indexer) }}
    }
}
if it is a function call operation then the structure is:
{
    "type": "functionCall",
    "data": {
        "callee": "{{ it can be either an identifier Val or an complex expression ( i mean a calculation ) that will finally resolved into a simple Val which is an identifier or directly a runnable function Val }}",
        "args": {{ an array of calculations in which each calculation can be a arithmetic type, or a host call, one of Val types, or another functionCall type or an indexer }},
    }
}

4. functionDefinition:

detect the function definition inside the input which i give you and put it in a
structure like below and just output it only:
{
    "type": "functionDefinition",
    "data": {
        "name": "{{ the name of the function to be defined which must be a raw string. for example "print" }}",
        "params": {{ an arrray of raw strings which are the name of the parameters of the function. for example ["num1", "textToPrint"] }},
        "body": {{ an arrray of other operations with mentioned types before }}
    }
}

5. conditionalBranch:

detect the conditional barnch inside the input which i give you and put it in a
structure like below and just output it only:
{
    "type": "conditionalBranch",
    "data": {
        "condition": "{{ can be an expression (basic or complex) which can consist of a nested operation between Val types, functionCalls or arithmetics or a host call, or an indexer. it must be resolvable into a bool Val type }}",
        "trueBranch": {{ the step number which program execution pointer must move to if the condition leads to true. it must be an integer, if the program should exit current function then the step number must be 0 }},
        "falseBranch": {{ the step number which program execution pointer must move to if the condition leads to false. it must be an integer, if the program should exit current function then the step number must be 0 }}
    }
}

6. jump operation:

detect the jump operation of the input which i give you and put it in a structure like below and just output it only:
{
    "type": "jumpOperation",
    "data": {
        "stepNumber": {{ the step number which program execution pointer must move to from this step, if the program should exit current function then the step number must be 0 }}
    }
}

7. return operation:

detect the returning details inside the input which i give you and put it in a
structure like below and just output it only:
{
    "type": "returnOperation",
    "data": {
        "value": "{{ can be an expression (basic or complex) which can consist of a nested operation between Val types, functionCalls or arithmetics or a host call, or an indexer like explained before }}",
    }
}

8. function call operation:

if it is a standalone function call then detect the function call details inside the input which i give you and put it in a
structure like below and just output it only:
{
    "type": "functionCall",
    "data": {
        "callee": "{{ it can be either an identifier Val or an complex expression ( i mean a calculation ) that will finally resolved into a simple Val which is an identifier or directly a runnable function Val }}",
        "args": {{ an array of calculations in which each calculation can be a arithmetic type, one of Val types, or another functionCall type or a host call, or an indexer }},
    }
}

9. host api call:

if it is a host api call then detect the call details inside the input which i give you and put it in a
structure like below and just output it only:
{
    "type": "host_call",
    "data": {
        "name": "{{ it is a raw string which is name of host api to be called. it can only be one of these values:
            - "println" to print something. it gets a string Val as arg.
            - "stringify" to convert a non string value to string. it gets a Val of different types as arg.
            - "getInput" to get an input string from user. it does not need arg.
            - "hasProp" to check if an object contains a property. its result is a bool Val. it gets 2 args: first arg is the object to
              investigated and the second arg can be an expression that must be able to be resolved to a string Val which is name of the
              property to be checked. 
            - "typeof" to get type name of a Val or an expression being resolved to Val, its return type is a string a Val too.
              the result string Val's raw string data can be one of "i16", "132", "i64", "f32", "f64", "bool", "string", "object", "array".
          and nothing else unless the user specifically asks to call a host api with a name and passing the argument
        }}",
        "args": {{ an array Val of calculations in which each calculations can be a arithmetic type, one of Val types, or a host call, or an functionCall type or an indexer }},
    }
}

10. prompt-based-code-generation:

if it is a request for dynamically generating ast code at compile time. then do it. its rseult can be any of the previous AST types or even
a complex mix of them.

so you must generate a second output showing a custom AST that inside that, each step in program code with specified type in previous step is mapped to a
complex AST code step. remember that sub-steps of a step are assembled into a single complex AST step.

now the user will give you the code and you should first try to build a structured step operation types as shown in the example before and then 
based on the step operation types details, generate a structured custom AST using the instructions mentioned before and only print the AST json string as output.

only output the custom AST json code based on the previously defined rules:
{
    "type": "program",
    "body": [
        ?
    ]
}`

func processPacket(callbackId int64, data []byte) {
	defer func() {
		r := recover()
		if r != nil {
			var err error
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("unknown error")
			}
			log.Println(err)
		}
	}()
	if callbackId == 0 {
		if string(data) == "{}" {
			return
		}
		packet := map[string]any{}
		err := json.Unmarshal(data, &packet)
		if err != nil {
			log.Println(err)
			return
		}
		data := packet["data"].(string)
		input := map[string]any{}
		err = json.Unmarshal([]byte(data), &input)
		if err != nil {
			log.Println(err)
			return
		}
		userId := packet["user"].(map[string]any)["id"].(string)
		pointId := packet["point"].(map[string]any)["id"].(string)

		switch input["type"] {
		case "textMessage":
			message := strings.Trim(input["text"].(string), " ")
			if strings.HasPrefix(message, "@gemini") {
				inp := ListPointAppsInput{
					PointId: pointId,
				}
				submitOffchainBaseTrx(pointId, "/points/listApps", "", "", "", "", inp, func(pointsAppsRes []byte) {
					log.Println(string(pointsAppsRes))
					out := ListPointAppsOutput{}
					log.Println("parsing...")
					err := json.Unmarshal(pointsAppsRes, &out)
					if err != nil {
						log.Println(err.Error())
						signalPoint("single", pointId, userId, map[string]any{"type": "textMessage", "text": "an error happended " + string(pointsAppsRes)}, true)
					}
					log.Println("parsed.")

					machinesMeta := []map[string]any{}
					log.Println("starting to extract...")
					for k, v := range out.Machines {
						log.Println(k)
						if v.Identifier == "0" {
							if isMcpRaw, ok := v.Metadata["isMcp"]; ok {
								if isMcp, ok := isMcpRaw.(bool); ok && isMcp {
									log.Println(k + " is mcp")
									machinesMeta = append(machinesMeta, map[string]any{
										"machineId": v.UserId,
										"metadata":  v.Metadata,
									})
								}
							}
						}
					}

					toolsList := []*genai.FunctionDeclaration{}

					toolToMachIdMap := map[string]string{}

					for _, metaRaw := range machinesMeta {
						machineId := metaRaw["machineId"].(string)
						metaObj := metaRaw["metadata"].(map[string]any)
						tools := metaObj["tools"].([]any)
						for _, toolRaw := range tools {
							toolObj := toolRaw.(map[string]any)
							params := map[string]*genai.Schema{}
							for k, v := range toolObj["args"].(map[string]any) {
								t := v.(map[string]any)["type"]
								var typ genai.Type
								switch t {
								case "STRING":
									typ = genai.TypeString
								case "NUMBER":
									typ = genai.TypeNumber
								case "BOOL":
									typ = genai.TypeBoolean
								default:
									typ = genai.TypeUnspecified
								}
								params[k] = &genai.Schema{
									Title:       k,
									Type:        typ,
									Description: v.(map[string]any)["desc"].(string),
								}
							}
							toolName := toolObj["name"].(string)
							toolsList = append(toolsList, &genai.FunctionDeclaration{
								Name: toolName,
								Parameters: &genai.Schema{
									Type:       genai.TypeObject,
									Properties: params,
								},
								Description: toolObj["desc"].(string),
							})
							toolToMachIdMap[toolName] = machineId
						}
					}

					temp := strings.ReplaceAll(message, "\n", "")
					temp = strings.ReplaceAll(temp, "\t", "")
					temp = strings.Trim(temp, " ")

					if temp == "/reset" {
						history := []*genai.Content{}
						chatObj, ok := chats[pointId]
						if !ok {
							chatObj = &Chat{Key: pointId, History: history}
							chats[pointId] = chatObj
						}
						chatObj.History = []*genai.Content{}
						signalPoint("broadcast", pointId, "-", map[string]any{"type": "textMessage", "text": "context reset"})
						return
					}

					ctx := context.Background()
					client, err := genai.NewClient(ctx, &genai.ClientConfig{
						APIKey:  API_KEY,
						Backend: genai.BackendGeminiAPI,
					})
					if err != nil {
						log.Println(err)
					}

					history := []*genai.Content{}

					chatObj, ok := chats[pointId]
					if ok {
						history = chatObj.History
					} else {
						chatObj = &Chat{Key: pointId, History: history}
						chats[pointId] = chatObj
					}
					chatObj.History = append(chatObj.History, genai.NewContentFromText(message, genai.RoleUser))

					log.Println(toolsList)

					chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
						Tools: []*genai.Tool{
							{
								FunctionDeclarations: toolsList,
							},
						},
					}, history)

					res, _ := chat.SendMessage(ctx, genai.Part{Text: message})

					fc := res.FunctionCalls()
					if len(fc) > 0 {
						toolName := fc[0].Name
						args := fc[0].Args
						id := fc[0].ID
						toolCallbacks[pointId] = func(result map[string]any) []byte {
							history := []*genai.Content{}
							chatObj, ok := chats[pointId]
							if ok {
								history = chatObj.History
							} else {
								chatObj = &Chat{Key: pointId, History: history}
								chats[pointId] = chatObj
							}
							chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
								Tools: []*genai.Tool{
									{
										FunctionDeclarations: toolsList,
									},
								},
							}, history)
							res, _ = chat.SendMessage(ctx, genai.Part{FunctionResponse: &genai.FunctionResponse{ID: id, Name: toolName, Response: result}})
							response := res.Candidates[0].Content.Parts[0].Text
							chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
							return []byte(response)
						}
						point := Point{Id: pointId, PendingUserId: userId}
						point.Push()
						machId := toolToMachIdMap[toolName]
						log.Println(res)
						signalPoint("single", pointId, machId, map[string]any{"name": toolName, "args": args, "type": "execute", "machineId": toolToMachIdMap[toolName]}, true)
					} else if len(res.Candidates) > 0 {
						response := res.Candidates[0].Content.Parts[0].Text
						chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
						signalPoint("broadcast", pointId, "-", map[string]any{"type": "textMessage", "text": response})
					}
				})
			}
		case "mcpCallback":

			params := map[string]any{}
			json.Unmarshal([]byte(input["payload"].(string)), &params)

			log.Println(params)

			cb := toolCallbacks[pointId]
			if cb != nil {
				output := cb(params)
				delete(toolCallbacks, pointId)
				signalPoint("broadcast", pointId, "-", map[string]any{"type": "textMessage", "text": string(output)})
			}
		case "generate":
			ctx := context.Background()
			client, err := genai.NewClient(ctx, &genai.ClientConfig{
				APIKey:  API_KEY,
				Backend: genai.BackendGeminiAPI,
			})
			if err != nil {
				log.Fatal(err)
			}

			res, _ := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(input["text"].(string)), nil)

			if len(res.Candidates) > 0 {
				response := res.Candidates[0].Content.Parts[0].Text
				signalPoint("single", pointId, userId, map[string]any{"type": "textMessage", "text": response})
			}
		case "compileApp":
			ctx := context.Background()
			client, err := genai.NewClient(ctx, &genai.ClientConfig{
				APIKey:  API_KEY,
				Backend: genai.BackendGeminiAPI,
			})
			if err != nil {
				log.Fatal(err)
			}

			temp := float32(0)

			res, _ := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(input["text"].(string)),
				&genai.GenerateContentConfig{
					Temperature:       &temp,
					SystemInstruction: genai.NewContentFromText(INSTRCUTIONS, genai.RoleModel),
				})

			if len(res.Candidates) > 0 {
				response := res.Candidates[0].Content.Parts[0].Text
				response = response[len("```json"):]
				response = response[:len("```")]
				signalPoint("single", pointId, userId, map[string]any{"type": "compileAppRes", "ast": response})
			}
		}

	} else {
		cb := callbacks[callbackId]
		cb(data)
		delete(callbacks, callbackId)
	}
}

var lock sync.Mutex
var conn net.Conn

func main() {

	log.Println("started storag machine.")

	var err error
	conn, err = net.Dial("tcp", "10.10.0.3:8084")
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()
	log.Println("Container client connected")

	r := bufio.NewReader(conn)
	for {
		var ln uint32
		if err := binary.Read(r, binary.LittleEndian, &ln); err != nil {
			if err != io.EOF {
				log.Printf("read len err: %v", err)
			}
			os.Exit(0)
		}
		var callbackId uint64
		if err := binary.Read(r, binary.LittleEndian, &callbackId); err != nil {
			if err != io.EOF {
				log.Printf("read len err: %v", err)
			}
			os.Exit(0)
		}
		buf := make([]byte, ln)
		if _, err := io.ReadFull(r, buf); err != nil {
			log.Printf("read body err: %v", err)
			os.Exit(0)
		}
		log.Printf("recv: %s", string(buf))
		go func() {
			processPacket(int64(callbackId), buf)
		}()
	}
}

var cbCounter = int64(0)

func writePacket(data []byte, callback func([]byte)) {
	lock.Lock()
	defer lock.Unlock()
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
	conn.Write(lenBytes)
	if callback == nil {
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(0))
		conn.Write(cbId)
	} else {
		cbCounter++
		callbackId := cbCounter
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(cbCounter))
		callbacks[callbackId] = callback
		conn.Write(cbId)
	}
	conn.Write(data)
}

func signalPoint(typ string, pointId string, userId string, data any, temp ...bool) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	if len(temp) > 0 {
		isTemp := "false"
		if temp[0] {
			isTemp = "true"
		}
		packet, _ := json.Marshal(map[string]any{"key": "signalPoint", "input": map[string]any{
			"type":    typ + "|" + isTemp,
			"pointId": pointId,
			"userId":  userId,
			"data":    string(dataBytes),
		}})
		writePacket(packet, nil)
	} else {
		packet, _ := json.Marshal(map[string]any{"key": "signalPoint", "input": map[string]any{
			"type":    typ + "|false",
			"pointId": pointId,
			"userId":  userId,
			"data":    string(dataBytes),
		}})
		writePacket(packet, nil)
	}
}

func submitOffchainBaseTrx(pointId string, key string, requesterUserId string, requesterSignature string, tokenId string, tag string, input any, cb func([]byte)) {
	inp, _ := json.Marshal(input)
	packet, _ := json.Marshal(map[string]any{"key": "submitOnchainTrx", "input": map[string]any{
		"targetMachineId":    "-",
		"isRequesterOnchain": false,
		"key":                pointId + "|" + key + "|" + requesterUserId + "|" + requesterSignature + "|" + tokenId + "|" + "false" + "|",
		"pointId":            pointId,
		"isFile":             false,
		"isBase":             true,
		"tag":                tag,
		"packet":             string(inp),
	}})
	writePacket(packet, func(output []byte) {
		cb(output)
	})
}

func dbPutObj(typ string, objId string, obj map[string]string, cb func()) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "putObj",
		"objType": typ,
		"objId":   objId,
		"obj":     obj,
	}})
	writePacket(packet, func([]byte) {
		cb()
	})
}

func dbGetObj(typ string, objId string, cb func(map[string][]byte)) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "getObj",
		"objType": typ,
		"objId":   objId,
	}})
	writePacket(packet, func(b []byte) {
		result := map[string][]byte{}
		json.Unmarshal(b, &result)
		cb(result)
	})
}

func dbPutLink(linkKey string, linkVal string) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":  "putLink",
		"key": linkKey,
		"val": linkVal,
	}})
	writePacket(packet, nil)
}

func dbGetObjsByPrefix(objType string, prefix string, offset int, count int, cb func(map[string]map[string][]byte)) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "getObjsByPrefix",
		"objType": objType,
		"prefix":  prefix,
		"offset":  offset,
		"count":   count,
	}})
	writePacket(packet, func(b []byte) {
		result := map[string]map[string][]byte{}
		json.Unmarshal(b, &result)
		cb(result)
	})
}

func dbGetObjs(typ string, offset int, count int, cb func(map[string]map[string][]byte)) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "getObjs",
		"objType": typ,
		"offset":  offset,
		"count":   count,
	}})
	writePacket(packet, func(b []byte) {
		result := map[string]map[string][]byte{}
		json.Unmarshal(b, &result)
		cb(result)
	})
}
