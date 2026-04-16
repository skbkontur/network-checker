# network-checker
Эта утилита позволяет выполнить небольшую сетевую диагностику для выявления сетевых проблем

Что делает:
* Собирает локальные сетевые настройки хоста, на котором запущена:
  - публичный IP-адрес
  - информацию о сетевых интерфейсах
  - информацию о маршрутах
* Разрешает FQDN в IP-адрес для запрошенной цели (если цель передана как FQDN)
* Отображает локальный маршрут для запрошенной цели
* Проверяет доступность транспорта на основе TCP-проверок для запрошенных портов
* Отображает расширенную трассировку с TCP-пробами на основе отправки 1 пакета в секунду на протяжении 50 секунд (для работы этого функционала необходимы привилегии пользователя *root*)

# Установка

## Готовый бинарный файл

Скомпилированные бинарные файлы доступны в разделе [releases](https://github.com/skbkontur/network-checker/releases) данного проекта.

Пример установки последней доступной версии утилиты **network-checker** в текущий каталог:
```
curl -s https://api.github.com/repos/skbkontur/network-checker/releases/latest \
  | sed -n 's/.*"browser_download_url": //p' \
  | grep 'tar.gz' \
  | xargs wget -O network-checker.latest.tar.gz
tar -xzf network-checker.latest.tar.gz
```

## Сборка из исходного кода

Для самостоятельной сборки необходима среда разработки Golang версии не ниже, чем указана в [go.mod](go.mod).

Склонируйте данный git-репозиторий:

```
git clone https://github.com/skbkontur/network-checker.git
cd network-checker
```

Выполните сборку:

```
go build -o network-checker
```

Разрешите выполнение собранного бинарного файла:
```
chmod a+x network-checker
```

# Команды для запуска
```
./network-checker
sudo ./network-checker -destination="google.com" -port=80 -port=443
sudo ./network-checker -destination="8.8.8.8" -port=53
```
