// Утилита генерирует bcrypt-хэш пароля для ADMIN_PASSWORD_HASH:
//
//	go run ./cmd/hashpassword            — интерактивно, пароль скрыт (без эха)
//	echo "пароль" | go run ./cmd/hashpassword -stdin  — из stdin (скрипты)
//
// В обоих режимах пароль не попадает в историю shell и в аргументы процессов.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	fromStdin := flag.Bool("stdin", false, "читать пароль из stdin (первая строка) вместо скрытого ввода")
	flag.Parse()

	var password []byte
	if *fromStdin {
		line, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil && len(line) == 0 {
			log.Fatalf("Ошибка чтения пароля из stdin: %v", err)
		}
		// Срезаем завершающий перевод строки (LF и CRLF).
		password = trimNewline(line)
	} else {
		fmt.Print("Пароль: ")
		var err error
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatalf("Ошибка чтения пароля: %v", err)
		}
		fmt.Println()
	}

	if len(password) < 8 {
		log.Fatal("Пароль короче 8 символов")
	}

	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Ошибка хэширования: %v", err)
	}

	fmt.Printf("ADMIN_PASSWORD_HASH=%s\n", hash)
}

func trimNewline(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
