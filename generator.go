package main

import (
	"strconv"
	"strings"

	"bytes"

	"fmt"

	"path"

	"sort"

	"log"
	"os"

	google_protobuf "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

const (
	targetCmd     string = "cmd"
	targetPackMsg string = "pack"
	targetUnpack  string = "unpack"
	targetAs      string = "as"
	targetJava    string = "java"
	targetTS      string = "ts"
	targetTSPB    string = "ts.pb"
	targetTSModel string = "ts.model"
)

// Generator the auto code generator
type Generator struct {
	Request      *plugin.CodeGeneratorRequest
	Response     *plugin.CodeGeneratorResponse
	Params       map[string]string
	typesMapping map[string]string
}

// NewGenerator create a new code generator
func NewGenerator() *Generator {
	g := new(Generator)
	g.Request = new(plugin.CodeGeneratorRequest)
	g.Response = new(plugin.CodeGeneratorResponse)
	g.Params = make(map[string]string)
	g.typesMapping = make(map[string]string)
	g.typesMapping["TYPE_DOUBLE"] = "float64"
	g.typesMapping["TYPE_FLOAT"] = "float32"
	g.typesMapping["TYPE_INT32"] = "int32"
	g.typesMapping["TYPE_INT64"] = "int64"
	g.typesMapping["TYPE_UINT32"] = "uint32"
	g.typesMapping["TYPE_UINT64"] = "uint64"
	g.typesMapping["TYPE_SINT32"] = "int32"
	g.typesMapping["TYPE_SINT64"] = "int64"
	g.typesMapping["TYPE_FIXED32"] = "uint32"
	g.typesMapping["TYPE_FIXED64"] = "uint64"
	g.typesMapping["TYPE_SFIXED32"] = "int32"
	g.typesMapping["TYPE_SFIXED64"] = "int64"
	g.typesMapping["TYPE_BOOL"] = "bool"
	g.typesMapping["TYPE_STRING"] = "string"
	g.typesMapping["TYPE_BYTES"] = "[]byte"
	return g
}

// LoadParams load params from request
func (g *Generator) LoadParams() {
	for _, v := range strings.Split(g.Request.GetParameter(), ",") {
		if i := strings.Index(v, "="); i < 0 {
			g.Params[v] = "true"
		} else {
			g.Params[v[0:i]] = v[i+1:]
		}
	}
}

// GenerateFiles Generate Entrance
func (g *Generator) GenerateFiles() {
	flags := make([]bool, 8)
	_, flags[0] = g.Params[targetAs]
	_, flags[1] = g.Params[targetCmd]
	_, flags[2] = g.Params[targetPackMsg]
	_, flags[3] = g.Params[targetUnpack]
	_, flags[4] = g.Params[targetJava]
	_, flags[5] = g.Params[targetTS]
	_, flags[6] = g.Params[targetTSPB]
	_, flags[7] = g.Params[targetTSModel]

	filesToGen := 0
	for _, v := range flags {
		if v {
			filesToGen++
		}
	}

	if filesToGen == 0 {
		log.Println("please specify which files to be generated, candidates: cmd,pack,unpack or as")
		os.Exit(1)
	}

	g.Response.File = make([]*plugin.CodeGeneratorResponse_File, len(g.Request.ProtoFile)*filesToGen)
	responseFileIndex := 0
	for _, file := range g.Request.ProtoFile {
		sort.Sort(ByMsgTypeName(file.MessageType))
		if flags[1] { // generate cmd file
			g.Response.File[responseFileIndex] = g.generateCmdFile(file)
			responseFileIndex++
		}

		if flags[2] { // generate pack file
			g.Response.File[responseFileIndex] = g.generatePackFile(file)
			responseFileIndex++
		}

		if flags[3] { // generate unpack file
			g.Response.File[responseFileIndex] = g.generateUnpackFile(file)
			responseFileIndex++
		}

		if flags[0] { // generate as file
			g.Response.File[responseFileIndex] = g.generateAsFile(file)
			responseFileIndex++
		}

		if flags[4] { // generate java file
			g.Response.File[responseFileIndex] = g.generateJavaFile(file)
			responseFileIndex++
		}

		if flags[5] { // generate ts file
			g.Response.File[responseFileIndex] = g.generateTSFile(file)
			responseFileIndex++
		}

		if flags[6] { // generate ts proto builder file
			g.Response.File[responseFileIndex] = g.generateTSProtoBuilderFile(file)
			responseFileIndex++
		}

		if flags[7] { // generate ts proto model file
			g.Response.File[responseFileIndex] = g.generateTSProtoModelFile(file)
			responseFileIndex++
		}
	}
}

/*

g.typesMapping["TYPE_DOUBLE"] = "float64"
	g.typesMapping["TYPE_FLOAT"] = "float32"
	g.typesMapping["TYPE_INT32"] = "int32"
	g.typesMapping["TYPE_INT64"] = "int64"
	g.typesMapping["TYPE_UINT32"] = "uint32"
	g.typesMapping["TYPE_UINT64"] = "uint64"
	g.typesMapping["TYPE_SINT32"] = "int32"
	g.typesMapping["TYPE_SINT64"] = "int64"
	g.typesMapping["TYPE_FIXED32"] = "uint32"
	g.typesMapping["TYPE_FIXED64"] = "uint64"
	g.typesMapping["TYPE_SFIXED32"] = "int32"
	g.typesMapping["TYPE_SFIXED64"] = "int64"
	g.typesMapping["TYPE_BOOL"] = "bool"
	g.typesMapping["TYPE_STRING"] = "string"
	g.typesMapping["TYPE_BYTES"] = "[]byte"
*/
func (g *Generator) getTsTypesMapping(name string) string {
	switch name {
	case "TYPE_DOUBLE", "TYPE_FLOAT", "TYPE_INT32", "TYPE_INT64", "TYPE_UINT32", "TYPE_UINT64", "TYPE_SINT32",
		"TYPE_SINT64", "TYPE_FIXED32", "TYPE_FIXED64", "TYPE_SFIXED32", "TYPE_SFIXED64":
		return "number"

	case "TYPE_BOOL":
		return "bool"

	case "TYPE_STRING":
		return "string"

	case "TYPE_BYTES":
		return "byte"

	default:
		return ""

	}
}

func (g *Generator) isEnumType(name string, file *google_protobuf.FileDescriptorProto) bool {
	for _, v := range file.GetEnumType() {
		if v.GetName() == name {
			return true
		}
	}
	return false
}

func (g *Generator) isCmdType(name string) bool {
	s := strings.ToLower(name)
	return strings.HasSuffix(s, "request") ||
		strings.HasSuffix(s, "response") ||
		strings.HasSuffix(s, "event")
}

func (g *Generator) generateGoFileHeader(buf *bytes.Buffer, file *google_protobuf.FileDescriptorProto) {
	buf.WriteString("// Code generated by protoc-gen-gocmd.\n")
	buf.WriteString("// source: ")
	buf.WriteString(*file.Name)
	buf.WriteByte('\n')
	buf.WriteString("// DO NOT EDIT!\n")
	buf.WriteByte('\n')
	buf.WriteString("package ")
	var packageName string
	if *file.Options.GoPackage != "" {
		packageName = *file.Options.GoPackage
	} else {
		packageName = strings.Replace(*file.Package, ".", "_", -1)
	}
	buf.WriteString(packageName)
	buf.WriteByte('\n')
	buf.WriteByte('\n')
}

func (g *Generator) generateUnpackFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf := new(bytes.Buffer)
	g.generateGoFileHeader(buf, file)
	buf.WriteString("import \"fmt\"\n")
	buf.WriteByte('\n')
	buf.WriteString("func Unpack(fromCmd int32, data []byte) (interface{}, error) {\n")
	buf.WriteString(tab)
	buf.WriteString("switch fromCmd {\n")
	for _, msg := range file.GetMessageType() {
		typeName := strings.Title(msg.GetName())
		if !g.isCmdType(msg.GetName()) {
			continue
		}

		buf.WriteString(tab)
		buf.WriteString("case Cmd_")
		buf.WriteString(typeName)
		buf.WriteString(":\n")
		buf.WriteString(tab)
		buf.WriteString(tab)
		buf.WriteString("pb := new(")
		buf.WriteString(typeName)
		buf.WriteString(")\n")
		buf.WriteString(tab)
		buf.WriteString(tab)
		buf.WriteString("err := pb.Unmarshal(data)\n")
		buf.WriteString(tab)
		buf.WriteString(tab)
		buf.WriteString("return pb, err\n\n")
	}

	buf.WriteString(tab)
	buf.WriteString("default:\n")
	buf.WriteString(tab)
	buf.WriteString(tab)
	buf.WriteString("return nil, fmt.Errorf(\"unHandle cmd:%x\", fromCmd)\n")
	buf.WriteString(tab)
	buf.WriteString("}\n}")

	response := new(plugin.CodeGeneratorResponse_File)
	fileSuffix := path.Ext(*file.Name)
	generatedFileName := (*file.Name)[0:len(*file.Name)-len(fileSuffix)] + ".unpack.go"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) generatePackFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf := new(bytes.Buffer)
	isFirstMsg := true
	g.generateGoFileHeader(buf, file)
	//buf.WriteString("import (\n")
	//buf.WriteString(tab)
	//buf.WriteString("\"github.com/gogo/protobuf/proto\"\n")
	if file.GetSyntax() == "proto2" {
		buf.WriteString("import \"github.com/gogo/protobuf/proto\"\n\n")
	}
	//buf.WriteString(tab)
	//buf.WriteString("\"log\"\n")
	//buf.WriteString(")\n\n")

	for _, msg := range file.GetMessageType() {
		if !isFirstMsg {
			buf.WriteByte('\n')
		}

		isFirstArgument := true
		assignmentBuf := new(bytes.Buffer)
		msgTypeName := strings.Title(msg.GetName())

		buf.WriteString("func New")

		buf.WriteString(msgTypeName)
		buf.WriteByte('(')
		for _, field := range msg.GetField() {
			if !isFirstArgument {
				buf.WriteString(", ")
			}

			argumentName := strings.ToUpper(field.GetName()) //strings.Title(field.GetName())
			attributeName := strings.Title(field.GetName())
			buf.WriteString(argumentName)
			buf.WriteByte(' ')
			typeName, builtinType := g.typesMapping[field.GetType().String()]
			repeatedField := field.GetLabel().String() == "LABEL_REPEATED"
			var isEnumType bool
			if !builtinType {
				typeName = strings.Title(field.GetTypeName()[strings.LastIndex(field.GetTypeName(), ".")+1:])
				isEnumType = g.isEnumType(typeName, file)
				if !isEnumType {
					typeName = "*" + typeName
				}
			}

			if repeatedField {
				typeName = "[]" + typeName
			}

			buf.WriteString(typeName)

			assignmentBuf.WriteString(tab)
			assignmentBuf.WriteString(tab)
			assignmentBuf.WriteString(attributeName)
			assignmentBuf.WriteString(": ")
			if builtinType && !strings.HasPrefix(typeName, "[]") {
				if file.GetSyntax() == "proto3" {
					assignmentBuf.WriteString(argumentName)
					assignmentBuf.WriteString(",\n")

				}
				if file.GetSyntax() == "proto2" {
					assignmentBuf.WriteString("proto.")
					assignmentBuf.WriteString(strings.Title(typeName))
					assignmentBuf.WriteByte('(')
					assignmentBuf.WriteString(argumentName)
					assignmentBuf.WriteString("),\n")
				}
			} else {
				if isEnumType {
					assignmentBuf.WriteString("&")
				}
				assignmentBuf.WriteString(argumentName)
				assignmentBuf.WriteString(",\n")
			}

			if isFirstArgument {
				isFirstArgument = false
			}
		}
		buf.WriteString(") *")
		buf.WriteString(msgTypeName)
		buf.WriteString(" {\n")

		buf.WriteString(tab)
		buf.WriteString("return &")
		buf.WriteString(msgTypeName)
		if assignmentBuf.Len() > 0 {
			buf.WriteString("{\n")
			buf.WriteString(assignmentBuf.String())
			buf.WriteString(tab)
			buf.WriteString("}\n")
		} else {
			buf.WriteString("{}\n")
		}
		buf.WriteString("}\n")

		// generate marshal code
		buf.WriteString("func (m *")
		buf.WriteString(msgTypeName)
		buf.WriteString(") Bytes() []byte {\n")
		buf.WriteString(tab)
		buf.WriteString("data, _ := m.Marshal()\n")
		buf.WriteString(tab)
		//buf.WriteString("return data")
		//buf.WriteString("if data, err := proto.Marshal(m); err != nil {\n")
		//buf.WriteString(tab)
		//buf.WriteString(tab)
		//buf.WriteString("log.Println(")
		//buf.WriteString("\"failed to marshal ")
		//buf.WriteString(msgTypeName)
		//buf.WriteString(" due to error \", err.Error())\n")
		//buf.WriteString(tab)
		//buf.WriteString(tab)
		//buf.WriteString("return nil\n")
		//buf.WriteString(tab)
		//buf.WriteString("} else {\n")
		//buf.WriteString(tab)
		//buf.WriteString(tab)
		buf.WriteString("return data\n")
		buf.WriteString("}\n")

		if isFirstMsg {
			isFirstMsg = false
		}
	}

	response := new(plugin.CodeGeneratorResponse_File)
	fileSuffix := path.Ext(*file.Name)
	generatedFileName := (*file.Name)[0:len(*file.Name)-len(fileSuffix)] + ".pack.go"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) generateCmdFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	buf := new(bytes.Buffer)
	g.generateGoFileHeader(buf, file)
	messageCount := len(file.MessageType)
	messageIDOffset := 0x1000
	for messageIDOffset < messageCount {
		messageIDOffset <<= 4
	}

	messageID := messageIDOffset + 1
	for _, v := range file.GetMessageType() {
		if !g.isCmdType(v.GetName()) {
			continue
		}
		buf.WriteString("const Cmd_")
		buf.WriteString(strings.Title(v.GetName()))
		buf.WriteString(fmt.Sprintf(" = 0x%X\n", messageID))
		messageID++
	}

	response := new(plugin.CodeGeneratorResponse_File)
	fileSuffix := path.Ext(*file.Name)
	generatedFileName := (*file.Name)[0:len(*file.Name)-len(fileSuffix)] + ".cmd.go"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) generateAsFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	buf := new(bytes.Buffer)
	ns, hasNs := g.Params["asns"]
	if !hasNs {
		ns = file.GetPackage()
	}

	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf.WriteString("// Code generated by protoc-gen-gocmd.\n")
	buf.WriteString("// source: ")
	buf.WriteString(*file.Name)
	buf.WriteByte('\n')
	buf.WriteString("// DO NOT EDIT!\n")
	buf.WriteByte('\n')
	buf.WriteString("package ")
	buf.WriteString(ns)
	buf.WriteString("\n{\n")
	buf.WriteString(tab)
	buf.WriteString("public class ProtocolType{\n")
	messageCount := len(file.MessageType)
	messageIDOffset := 0x1000
	for messageIDOffset < messageCount {
		messageIDOffset <<= 4
	}

	messageID := messageIDOffset + 1
	for _, msg := range file.GetMessageType() {
		if !g.isCmdType(msg.GetName()) {
			continue
		}
		buf.WriteString(tab)
		buf.WriteString(tab)
		buf.WriteString("public static const ")
		buf.WriteString(strings.Title(msg.GetName()))
		buf.WriteString(fmt.Sprintf(" : int = 0x%X;\n", messageID))
		messageID++
	}

	buf.WriteString(tab)
	buf.WriteString("}\n}\n")
	response := new(plugin.CodeGeneratorResponse_File)
	generatedFileName := "ProtocolType.as"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) generateTSFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	buf := new(bytes.Buffer)

	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf.WriteString("// Code generated by protoc-gen-gocmd.\n")
	buf.WriteString("// source: ")
	buf.WriteString(*file.Name)
	buf.WriteByte('\n')
	buf.WriteString("// DO NOT EDIT!\n")
	buf.WriteByte('\n')
	buf.WriteString("module proto.cmd {\n")
	messageCount := len(file.MessageType)
	messageIDOffset := 0x1000
	for messageIDOffset < messageCount {
		messageIDOffset <<= 4
	}

	messageID := messageIDOffset + 1
	for _, msg := range file.GetMessageType() {
		if !g.isCmdType(msg.GetName()) {
			continue
		}
		buf.WriteString(tab)
		buf.WriteString("export var Cmd_")
		buf.WriteString(strings.Title(msg.GetName()))
		buf.WriteString(fmt.Sprintf(": number = 0x%X;\n", messageID))
		messageID++
	}

	buf.WriteString("}\n")
	response := new(plugin.CodeGeneratorResponse_File)
	generatedFileName := "proto.cmd.ts"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) generateTSProtoBuilderFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	buf := new(bytes.Buffer)

	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf.WriteString("// Code generated by protoc-gen-gocmd.\n")
	buf.WriteString("// source: ")
	buf.WriteString(*file.Name)
	buf.WriteByte('\n')
	buf.WriteString("// DO NOT EDIT!\n")
	buf.WriteByte('\n')
	buf.WriteString("module proto {\n")

	for _, msg := range file.GetMessageType() {
		if !g.isCmdType(msg.GetName()) {
			continue
		}
		buf.WriteString(tab)
		buf.WriteString("export var ")
		buf.WriteString(msg.GetName())
		buf.WriteString(" = { cmd: proto.cmd.Cmd_")
		buf.WriteString(msg.GetName())
		buf.WriteString(", cls: \"proto.builder.")
		buf.WriteString(msg.GetName())
		buf.WriteString("\"")
		if msg.GetName() == "LoginResponse" {
			buf.WriteString(", auto_listen: false")
		}
		buf.WriteString("};\n")
	}

	buf.WriteString("}\n")
	response := new(plugin.CodeGeneratorResponse_File)
	generatedFileName := "proto.builder.ts"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

// todo
/*

module proto.model {


	//获取游戏配置响应
	export class GetGameConfigResponse {
		public roomId: string;
		public amount: string;
		public seat: number;
		public configs: Array<GameConfig>; // 游戏配置列表
	}

	export enum CODE {
		SUCCESS = 0,
	}
}
*/
func (g *Generator) generateTSProtoModelFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	buf := new(bytes.Buffer)

	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf.WriteString("// Code generated by protoc-gen-gocmd.\n")
	buf.WriteString("// source: ")
	buf.WriteString(*file.Name)
	buf.WriteByte('\n')
	buf.WriteString("// DO NOT EDIT!\n")
	buf.WriteByte('\n')
	buf.WriteString("module proto.model {\n")

	for _, enumType := range file.GetEnumType() {
		buf.WriteString(tab)
		buf.WriteString("export enum ")
		buf.WriteString(enumType.GetName())
		buf.WriteString(" {\n")
		for _, enumElement := range enumType.GetValue() {
			buf.WriteString(tab)
			buf.WriteString(tab)
			buf.WriteString(enumElement.GetName())
			buf.WriteString(" = ")
			buf.WriteString(strconv.Itoa(int(enumElement.GetNumber())))
			buf.WriteString(",")
			buf.WriteString("\n")
		}
		buf.WriteString(tab)
		buf.WriteString("}\n\n")

	}

	for _, msg := range file.GetMessageType() {
		if msg.GetName() == "RequestMessage" {
			continue
		}
		if msg.GetName() == "ResponseMessage" {
			continue
		}
		buf.WriteString(tab)
		buf.WriteString("export class ")
		buf.WriteString(msg.GetName())
		buf.WriteString(" {\n")

		for _, field := range msg.GetField() {
			buf.WriteString(tab)
			buf.WriteString(tab)
			buf.WriteString("public ")
			buf.WriteString(field.GetName())
			buf.WriteString(": ")

			tsTypeName := g.getTsTypesMapping(field.GetType().String())
			if tsTypeName == "" {
				tsTypeName = field.GetTypeName()[strings.LastIndex(field.GetTypeName(), ".")+1:]
			}
			if field.GetLabel().String() == "LABEL_REPEATED" {
				buf.WriteString("Array<")
				buf.WriteString(tsTypeName)
				buf.WriteString(">")
			} else {
				buf.WriteString(tsTypeName)
			}
			buf.WriteString(";\n")
		}
		buf.WriteString(tab)
		buf.WriteString("}\n\n")

	}

	buf.WriteString("}\n")
	response := new(plugin.CodeGeneratorResponse_File)
	generatedFileName := "proto.model.ts"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) generateJavaFile(file *google_protobuf.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
	buf := new(bytes.Buffer)
	pkg, hasPkg := g.Params["pkg"]
	if !hasPkg {
		pkg = file.GetPackage()
	}

	tab := "    " // 4 spaces for tab by default
	if _, ok := g.Params["usetabs"]; ok {
		tab = "\t"
	}

	buf.WriteString("// Code generated by protoc-gen-gocmd.\n")
	buf.WriteString("// source: ")
	buf.WriteString(*file.Name)
	buf.WriteByte('\n')
	buf.WriteString("// DO NOT EDIT!\n")
	buf.WriteByte('\n')
	buf.WriteString("package ")
	buf.WriteString(pkg)
	buf.WriteString(";\n\n")
	buf.WriteString("import java.util.Map;\n")
	buf.WriteString("import java.util.HashMap;\n\n")
	buf.WriteString("public class MessageTypes {\n")
	messageCount := len(file.MessageType)
	messageIDOffset := 0x1000
	for messageIDOffset < messageCount {
		messageIDOffset <<= 4
	}

	messageID := messageIDOffset + 1
	for _, msg := range file.GetMessageType() {
		buf.WriteString(tab)
		buf.WriteString("public static final int ")
		buf.WriteString(strings.Title(msg.GetName()))
		buf.WriteString(fmt.Sprintf(" = 0x%X;\n", messageID))
		messageID++
	}

	buf.WriteString("\n")
	buf.WriteString(tab)
	buf.WriteString("private static Map<Integer, String> messageTypeToMessageNameMapping = new HashMap<Integer, String>();\n")
	buf.WriteString("private static Map<String, Integer> messageNameToMessageTypeMapping = new HashMap<String, Integer>();\n\n")
	buf.WriteString(tab)
	buf.WriteString("static {\n")
	for _, msg := range file.GetMessageType() {
		buf.WriteString(tab)
		buf.WriteString(tab)
		buf.WriteString("messageTypeToMessageNameMapping.put(")
		buf.WriteString(strings.Title(msg.GetName()))
		buf.WriteString(", \"")
		buf.WriteString(msg.GetName())
		buf.WriteString("\");\n")
		buf.WriteString(tab)
		buf.WriteString(tab)
		buf.WriteString("messageNameToMessageTypeMapping.put(\"")
		buf.WriteString(msg.GetName())
		buf.WriteString("\", ")
		buf.WriteString(strings.Title(msg.GetName()))
		buf.WriteString(");\n")
	}
	buf.WriteString(tab)
	buf.WriteString("}\n\n")
	buf.WriteString(tab)
	buf.WriteString("public static String getMessageTypeName(int messageTypeId) {\n")
	buf.WriteString(tab)
	buf.WriteString(tab)
	buf.WriteString("return messageTypeToMessageNameMapping.get(messageTypeId);\n")
	buf.WriteString(tab)
	buf.WriteString("}\n\n")
	buf.WriteString(tab)
	buf.WriteString("public static Integer getMessageTypeId(String messageTypeName) {\n")
	buf.WriteString(tab)
	buf.WriteString(tab)
	buf.WriteString("return messageNameToMessageTypeMapping.get(messageTypeName);\n")
	buf.WriteString(tab)
	buf.WriteString("}\n")
	buf.WriteString("}\n")
	response := new(plugin.CodeGeneratorResponse_File)
	generatedFileName := "MessageTypes.java"
	fileContent := buf.String()
	response.Name = &generatedFileName
	response.Content = &fileContent
	return response
}

func (g *Generator) isMessageCandidate(name string) bool {
	suffixes, ok := g.Params["suffixes"]
	if !ok {
		return true
	}

	for _, v := range strings.Split(suffixes, "/") {
		if strings.HasSuffix(name, v) {
			return true
		}
	}

	return false
}

//ByMsgTypeName sort all message types by name, so the protoId will be the same for each type of message in different runs
type ByMsgTypeName []*google_protobuf.DescriptorProto

func (t ByMsgTypeName) Len() int {
	return len(t)
}

func (t ByMsgTypeName) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t ByMsgTypeName) Less(i, j int) bool {
	return strings.Compare(*t[i].Name, *t[j].Name) < 0
}
