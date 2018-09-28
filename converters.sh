_converters() {
    echo "package main"
    echo
    echo "import \"encoding/json\""
    for o in $(echo "int:0 int64:0 string:\"\""); do
        f=$(echo "$o" | cut -d ":" -f 1)
        d=$(echo "$o" | cut -d ":" -f 2)
        echo "
func ${f}Converter(expect $f, d []byte, op opType) bool {
    i, ok := ${f}FromJSON(d)
    if ok {
        switch op {"
        if [[ $d == "0" ]]; then
            echo "
        case lessThan:
            return i < expect
        case lessTE:
            return i <= expect
        case greatThan:
            return i > expect
        case greatTE:
            return i >= expect"
        fi
        echo "
        case nEquals:
            return i != expect
        case equals:
            return i == expect
        }
    }
    return false
}

func ${f}FromJSON(d []byte) ($f, bool) {
    var i $f
    err := json.Unmarshal(d, &i)
    if err != nil {
        return $d, false
    }
    return i, true
}"
    done
}

_converters | sed "s/    /\t/g" > cmd/generated.go
