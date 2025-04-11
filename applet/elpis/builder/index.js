let acorn = require("acorn");
let fs = require("fs");
let { Buffer } = require("node:buffer");

let code = fs.readFileSync("../sample.js", { encoding: "utf8" });

let ast = acorn.parse(code, { ecmaVersion: 2025 });

console.dir(ast, { depth: null, colors: true })

let intToBytes = (x) => {
    const bytes = Buffer.alloc(4);
    bytes.writeInt32BE(x);
    return bytes;
}

let longToBytes = (x) => {
    const bytes = Buffer.alloc(8);
    bytes.writeBigInt64BE(x);
    return bytes;
}

let stringToBytes = (x) => {
    const buffer = Buffer.from(x, 'utf-8');
    const bytes = Array.from(buffer);
    return [...intToBytes(bytes.length), ...bytes];
}

let valueToBytes = (x) => {
    if (typeof x == 'number') {
        return [0xf1, ...intToBytes(x)];
    } else if (typeof x === 'bigint') {
        return [0xf2, ...longToBytes(x)];
    } else if (typeof x === 'boolean') {
        return [0xf3, (x == true ? 0x01 : 0x00)];
    } else if (typeof x === 'string') {
        return [0xf4, ...stringToBytes(x)];
    } else if (typeof x === 'undefined') {
        return [0xf5, 0x00]
    }
}

let stack = [];

let resolveNodeValue = (node) => {
    let result = [];
    if (node.type === "Identifier") {
        result.push(0xf0);
        result.push(...stringToBytes(node.name));
    } else if (node.type === "Literal") {
        return valueToBytes(node.value);
    } else if (node.type === "ObjectExpression") {
        result.push(0xf6);
        result.push(...intToBytes(node.properties.length));
        node.properties.forEach(prop => {
            result.push(...stringToBytes(prop.key.name));
            result.push(...resolveNodeValue(prop.value));
        });
    } else if (node.type === "ArrayExpression") {
        result.push(0xf7);
        result.push(...intToBytes(node.elements.length));
        node.elements.forEach(item => {
            result.push(...resolveNodeValue(item));
        });
    } else if (node.type === "BinaryExpression") {
        result.push(0xf8);
        if (node.operator === "+") {
            result.push(0x01);
        } else if (node.operator === "-") {
            result.push(0x02);
        } else if (node.operator === "*") {
            result.push(0x03);
        } else if (node.operator === "/") {
            result.push(0x04);
        } else if (node.operator === "%") {
            result.push(0x05);
        } else if (node.operator === "^") {
            result.push(0x06);
        }
        result.push(...resolveNodeValue(node.left));
        result.push(...resolveNodeValue(node.right));
    } else if (node.type === "MemberExpression") {
        result.push(0xf9);
        result.push(...resolveNodeValue(node.object));
        result.push(...resolveNodeValue(node.property));
    } else if (node.type === "UpdateExpression") {
        result.push(0xfa);
        if (node.operator === "++") {
            result.push(0x01);
            result.push(...resolveNodeValue(node.argument));
        } else if (node.operator === "--") {
            result.push(0x02);
            result.push(...resolveNodeValue(node.argument));
        }
    } else if (node.type === "AssignmentExpression") {
        result.push(0xfb);
        if (node.operator === "=") {
            result.push(0x01);
            result.push(...stringToBytes(node.left.name));
            result.push(...resolveNodeValue(node.right));
        }
    } else if (node.type === "CallExpression") {
        let temp = [];
        temp.push(0xfc);
        temp.push(...resolveNodeValue(node.callee));
        temp.push(...intToBytes(node.arguments.length));
        node.arguments.forEach(a => {
            temp.push(...resolveNodeValue(a));
        });
        stack[stack.length - 1].temps.push(0x01);
        let tempId = "_temp_" + stack[stack.length - 1].tempCounter;
        stack[stack.length - 1].temps.push(...stringToBytes(tempId));
        stack[stack.length - 1].tempCounter++;
        stack[stack.length - 1].temps.push(...temp);
        result.push(0xf0);
        result.push(...stringToBytes(tempId));
    } else if (node.type === "BlockStatement") {
        let blockByteCode = blockToMachine(node.body);
        result.push(...intToBytes(blockByteCode.length));
        result.push(...blockByteCode);
    } else if (node.type === "ArrowFunctionExpression") {
        result.push(0xfe);
        result.push(...intToBytes(node.params.length));
        node.params.forEach(p => {
            result.push(...resolveNodeValue(p));
        });
        result.push(...resolveNodeValue(node.body));
    } else if (node.type === "FunctionExpression") {
        result.push(0xfe);
        result.push(...intToBytes(node.params.length));
        node.params.forEach(p => {
            result.push(...resolveNodeValue(p));
        });
        result.push(...resolveNodeValue(node.body));
    }
    return result;
}

let resolveClassBody = (body) => {
    let result = [];
    body.forEach(node => {
        if (node.type === "PropertyDefinition") {
            result.push(0x01);
            result.push(...stringToBytes(node.key.name));
            if (node.value === null) {
                result.push(...valueToBytes(undefined));
            } else {
                result.push(...resolveNodeValue(node.value));
            }
        } else if (node.type === "MethodDefinition") {
            result.push(0x02);
            result.push(...stringToBytes(node.key.name));
            if (node.value === null) {
                result.push(...valueToBytes(undefined));
            } else {
                result.push(...resolveNodeValue(node.value));
            }
        }
    });
    return result;
}

let toMachine = (node) => {
    let result = [];
    if (node.type === "VariableDeclaration") {
        node.declarations.forEach(d => {
            let temp = resolveNodeValue(d.init);
            result.push(...stack[stack.length - 1].temps);
            stack[stack.length - 1].temps = [];
            stack[stack.length - 1].tempCounter = 0;
            result.push(0x01);
            result.push(...stringToBytes(d.id.name));
            result.push(...temp);
        });
    } else if (node.type === "ExpressionStatement") {
        let temp = resolveNodeValue(node.expression);
        result.push(...stack[stack.length - 1].temps);
        stack[stack.length - 1].temps = [];
        stack[stack.length - 1].tempCounter = 0;
        result.push(0x02);
        result.push(...temp);
    } else if (node.type === "FunctionDeclaration") {
        let temp = [];
        node.params.forEach(p => {
            temp.push(...resolveNodeValue(p));
        });
        result.push(...stack[stack.length - 1].temps);
        stack[stack.length - 1].temps = [];
        stack[stack.length - 1].tempCounter = 0;
        let tempFunc = [];
        tempFunc.push(0x03);
        tempFunc.push(...stringToBytes(node.id.name));
        tempFunc.push(...intToBytes(node.params.length));
        tempFunc.push(...temp);
        tempFunc.push(...resolveNodeValue(node.body));
        stack[stack.length - 1].funcs.push(...tempFunc);
    } else if (node.type === "ClassDeclaration") {
        result.push(0x04);
        result.push(...stringToBytes(node.id.name));
        result.push(...resolveClassBody(node.body.body));
    } else if (node.type === "ReturnStatement") {
        let temp = resolveNodeValue(node.argument);
        result.push(...stack[stack.length - 1].temps);
        stack[stack.length - 1].temps = [];
        stack[stack.length - 1].tempCounter = 0;
        result.push(0x05);
        result.push(...temp);
    }
    return result;
}

let blockToMachine = (body) => {
    stack.push({ funcs: [], temps: [], tempCounter: 0 });
    let result = [];
    body.forEach(node => {
        result.push(...toMachine(node));
    });
    let funcs = stack.pop().funcs;
    return [...funcs, ...result];
}

machineCode = blockToMachine(ast.body);

console.log(machineCode);

// temp = [0x01, 0x00, 0x00, 0x00, 0x01, 97, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01]

// fs.writeFileSync("temp.elp", Buffer.from(temp));

console.log(Buffer.from(machineCode).toString('base64'));
