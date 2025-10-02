
function a(b) {
    return b + 1;
}

function b(text) {
    return "hello " + text;
}

function add(a, b) {
    return a + b;
}

console.log(b(add(a(1), a(2))));
