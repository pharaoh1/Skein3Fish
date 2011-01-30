package main

import (
    "fmt"
    "os"
    "bufio"
    "strings"
    "encoding/hex"
    "crypto/skein"
    "bytes"
    )

type katResult struct {
    stateSize int
    hashBitLength int
    msgLength int
    msg []byte
    msgFill int
    result []byte
    resultFill int
    macKeyLen int
    macKey []byte
    macKeyFill int
    restOfLine []byte
}

func checkKATVectors(ks *katScanner) bool {
    kr := new(katResult);
    
    var tree, mac, normal int

    for  ks.fillResult(kr) {
        if strings.Contains(string(kr.restOfLine), "Tree") {
            tree++
            continue
        }
        if strings.Contains(string(kr.restOfLine), "MAC") {
            mac++
            continue
        }
        fmt.Printf("state: %d, length: %d\n", kr.stateSize, kr.hashBitLength)
        skein := skein.NewSkein(kr.stateSize, kr.hashBitLength)
        skein.UpdateBits(kr.msg, 0, kr.msgLength)
        hash := skein.DoFinal()
        if ret := bytes.Compare(hash, kr.result); ret != 0 {
            fmt.Printf("Computed hash:\n%s\n", hex.EncodeToString(hash))
            fmt.Printf("Expected result:\n%s\n", hex.EncodeToString(kr.result))
            return false
        }
        // Enable the next few line so you can check some results manually
        // if ((kr.msgLength & 1) == 1) {
        // System.out.println(kr.stateSize + "-" + kr.hashBitLength + "-"
        // + kr.msgLength + "-" + kr.restOfLine);
        // hexdump("Computed hash", hash, hash.length);
        // hexdump("Expected result", kr.result, kr.result.length);
        // }
        normal++
    }
    fmt.Printf("Tree: %d, mac: %d, normal: %d\n", tree, mac, normal)
    return true
}

func setUp() *katScanner {
    scanner := newKatScanner("../../../data/skein_golden_kat.txt");
    if scanner == nil {
        return nil
    }
    return scanner
}

func vectorTest(ks *katScanner) {
    checkKATVectors(ks)
}


/*
 * Scanner for KAT file that fills in the KAT test vectors and
 * expected results.
 *
 */
const Start = 0
const Message = 1
const Result = 2
const MacKeyHeader = 3
const MacKey = 4
const Done = 5

type katScanner struct {
    buf   *bufio.Reader
    state int
}

func newKatScanner(name string) *katScanner {
    r, e := os.Open(name, os.O_RDONLY, 0)
    if e != nil {
        fmt.Printf("File open error: %s\n", e)
        return nil
    }
    bufio := bufio.NewReader(r);
    return &katScanner{bufio, Start}
}

/**
 * Fill in data from KAT file, one complete element at a time.
 * 
 * @param kr The resulting KAT data
 * @return
 */
func (s *katScanner) fillResult(kr *katResult) bool {

    dataFound := false;

    for s.state != Done {
        line, err := s.buf.ReadString('\n')
        if err != nil { 
            break
        }
        s.parseLines(line, kr);
        dataFound = true;
     }
     s.state = Start
     return dataFound
}

func (s *katScanner) parseLines(line string, kr *katResult) {
//    fmt.Printf("Line: %s", line)
    
    line = strings.TrimSpace(line)
    
    if len(line) <= 1 {
        return
    }
    
    if strings.HasPrefix(line, "Message") {
        s.state = Message
        return
    }
    if strings.HasPrefix(line, "Result") {
        s.state = Result
        return
    }
    if strings.HasPrefix(line, "MAC") {
        s.state = MacKeyHeader
    }
    if strings.HasPrefix(line, "------") {
        s.state = Done
        return
    }
    switch s.state {
        case Start:
        if strings.HasPrefix(line, ":Skein-") {
            s.parseHeaderLine(line, kr);
        } else {
            fmt.Printf("Wrong format found");
            os.Exit(1);
        }
        case Message:
            s.parseMessageLine(line, kr)
        case Result:
            s.parseResultLine(line, kr)
        case MacKey:
            s.parseMacKeyLine(line, kr)
        case MacKeyHeader:
            s.parseMacKeyHeaderLine(line, kr)
    }
}

func (s *katScanner) parseHeaderLine(line string, kr *katResult) {
    var rest string;

    ret, err := fmt.Sscanf(line, ":Skein-%d: %d-bit hash, msgLen = %d%s",
                 &kr.stateSize, &kr.hashBitLength, &kr.msgLength, &rest);
    if err != nil {
        fmt.Printf("state size: %d, bit length: %d, msg length: %d, rest: %s, ret: %d\n",
                    kr.stateSize, kr.hashBitLength, kr.msgLength, rest, ret)
    }

    idx := strings.Index(line, rest);
    kr.restOfLine = make([]byte, len(line) - idx)
    copy(kr.restOfLine[:], line[idx:])

    if kr.msgLength > 0 {
        if (kr.msgLength % 8) != 0 {
            kr.msg = make([]byte, (kr.msgLength >> 3) + 1)
        } else {
            kr.msg = make([]byte, kr.msgLength >> 3)
        }
    }
    if (kr.hashBitLength % 8) != 0 {
        kr.result = make([]byte, (kr.hashBitLength >> 3) + 1)
    } else {
        kr.result = make([]byte, kr.hashBitLength >> 3)
    }
    kr.msgFill = 0;
    kr.resultFill = 0;
    kr.macKeyFill = 0;

}

func (s *katScanner) parseMessageLine(line string, kr *katResult) {
    var d [16]int

    if strings.Contains(line, "(none)") {
        return;
    }
    ret, err := fmt.Sscanf(line,"%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x",
        &d[0], &d[1], &d[2], &d[3], &d[4], &d[5], &d[6], &d[7], &d[8], &d[9], &d[10], &d[11], &d[12], &d[13], &d[14], &d[15])

    for i := 0; i < ret; i++ {
        kr.msg[kr.msgFill] = byte(d[i])
        kr.msgFill++
    }
    if err != nil && ret <= 0 {
        fmt.Printf("msg: %s, ret: %d, %s \n", hex.EncodeToString(kr.msg), ret, err)
    }
}

func (s *katScanner) parseResultLine(line string, kr *katResult) {
    var d [16]int

    ret, err := fmt.Sscanf(line,"%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x",
        &d[0], &d[1], &d[2], &d[3], &d[4], &d[5], &d[6], &d[7], &d[8], &d[9], &d[10], &d[11], &d[12], &d[13], &d[14], &d[15])

    for i := 0; i < ret; i++ {
        kr.result[kr.resultFill] = byte(d[i])
        kr.resultFill++
    }
    if err != nil && ret <= 0 {
        fmt.Printf("result: %s, ret: %d, %s \n", hex.EncodeToString(kr.result))
    }
}

func (s *katScanner) parseMacKeyLine(line string, kr *katResult) {
    var d [16]int

    if strings.Contains(line, "(none)") {
        return;
    }
    ret, err := fmt.Sscanf(line,"%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x",
        &d[0], &d[1], &d[2], &d[3], &d[4], &d[5], &d[6], &d[7], &d[8], &d[9], &d[10], &d[11], &d[12], &d[13], &d[14], &d[15])

    for i := 0; i < ret; i++ {
        kr.macKey[kr.macKeyFill] = byte(d[i])
        kr.macKeyFill++
    }
    if err != nil && ret <= 0 {
        fmt.Printf("macKey: %s, ret: %d, %s \n", hex.EncodeToString(kr.macKey), ret, err)
    }
}

func (s *katScanner) parseMacKeyHeaderLine(line string , kr *katResult) {
    var rest string

    ret, err := fmt.Sscanf(line, "MAC key = %d%s", &kr.macKeyLen, &rest);

    if kr.macKeyLen != 0 && ret > 0 {
        kr.macKey = make([]byte, kr.macKeyLen);
    }
    if err != nil && ret <= 0 {
        fmt.Printf("macKeyLen: %d, %s\n", kr.macKeyLen, err)
    }
    s.state = MacKey;
}


