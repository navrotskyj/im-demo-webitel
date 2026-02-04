# Використовуємо офіційний образ Go
FROM golang:1.25.6-bookworm AS builder


RUN apt-get update && \
    apt-get install -y \
    build-essential \
    libssl-dev \
    ca-certificates

# Встановлюємо delve для дебагу
# Впевнюємося, що delve встановлюється для правильної архітектури.
RUN GOOS=linux GOARCH=arm64 go install github.com/go-delve/delve/cmd/dlv@latest

# Встановлюємо робочий каталог всередині контейнера
WORKDIR /app

# Копіюємо файли go.mod та go.sum для кешування залежностей
COPY go.mod go.sum ./
RUN go mod download

# Скопіюйте папку speechsdk ПЕРЕД КОПІЮВАННЯМ ВСІХ ІНШИХ ФАЙЛІВ
# Це важливо, щоб інклюди та бібліотеки були доступні на етапі компіляції.
# Припускається, що speechsdk знаходиться в корені вашого проекту.
#COPY speechsdk /app/speechsdk

COPY env /app/env

# Копіюємо *всі* інші файли з поточного каталогу на хост-машині
COPY . .

# *** НАЙВАЖЛИВІША ЗМІНА: Правильно встановлюємо CGO_CFLAGS та CGO_LDFLAGS ***
# CGO_CFLAGS для прапорів КОМПІЛЯТОРА C (наприклад, -I для інклюдів)
# CGO_LDFLAGS для прапорів КОМПОНУВАЛЬНИКА (наприклад, -L для шляхів бібліотек, -l для назв бібліотек)

# Ваші прапори для компілятора C.
#ENV CGO_CFLAGS="-I/app/speechsdk/include/c_api -Wall -Werror -fno-stack-protector -Wdeclaration-after-statement"

# Ваші прапори для компонувальника.
# Якщо -static не спрацює для зовнішньої бібліотеки, її доведеться копіювати.
#ENV CGO_LDFLAGS="-L/app/speechsdk/lib/arm64 -lMicrosoft.CognitiveServices.Speech.core"
# Змінив /x64 на /arm64, оскільки ви компілюєте для aarch64.
# Переконайтеся, що у вас є версія Speech SDK для ARM64.

# Компілюємо ваш додаток для Linux з увімкненим Cgo
# Встановлюємо GOOS та GOARCH для AArch64.
RUN GOOS=linux GOARCH=amd64 go build  -o /app/im-demo-service .

# --- Секція для виконання (для дебагу або запуску) ---
FROM debian:bullseye-slim

# Встановлюємо delve з першого етапу
COPY --from=builder /go/bin/dlv /usr/local/bin/dlv

WORKDIR /app

# Копіюємо скомпільований бінарний файл
COPY --from=builder /app/im-demo-service .

# Копіюємо також динамічні бібліотеки speechsdk, якщо вони не були статично скомпільовані.
# Це критично, якщо '-static' у CGO_LDFLAGS не повністю спрацював або якщо бібліотеки не підтримують повне статичне лінкування.
# Шлях для ARM64.
#COPY --from=builder /app/speechsdk/lib/arm64/libMicrosoft.CognitiveServices.Speech.core.so /usr/lib/

# Відкриваємо порт для delve
EXPOSE 2345
EXPOSE 50011

# Команда за замовчуванням
#CMD ["dlv", "--listen=:2345", "--headless=true", "--api-version=2",  "exec", "./im-demo-service", "server"]
