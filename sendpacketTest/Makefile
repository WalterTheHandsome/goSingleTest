
#
# makefile of CLI HAL C for agx5000
#

all: 
	make ping
	make listen

ping: ./pinger/main.go
	go build -o ping ./pinger/main.go

listen: ./listener/main.go
	go build -o listen ./listener/main.go

clean:
	rm ping
	rm listen
