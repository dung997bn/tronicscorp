
#STAGE1
#build stage
FROM golang:alpine AS builder
ENV GO111MODULE=on \
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64

# current working directory is /build in the container
WORKDIR /build

# copy over go.mod and go.sum (modules dependencies and checksum)
# over to working directory
COPY go.mod .
COPY go.sum .

#dowload dependencies
RUN go mod download

#copy application code to container
COPY . .


#building main
RUN go build main.go



#STAGE2
#Build a small image
FROM scratch

#agument to be passed during build phase
ARG MY_APP_PORT
ARG DB_HOST
ARG DB_PORT
ARG JWT_TOKEN_SECRET


#enviroment variables for the application
ENV MY_APP_PORT=${MY_APP_PORT}
ENV DB_HOST=${DB_HOST}
ENV DB_PORT=${DB_PORT}
ENV JWT_TOKEN_SECRET=${JWT_TOKEN_SECRET}

#copy from stage 1 image
COPY --from=builder build/main /

#label
LABEL Name=tronicscorp Version=0.0.1

#expose the port to run application
EXPOSE 8080

#command to run
ENTRYPOINT ["sh", "/main" ]




# RUN apk add --no-cache git
# RUN go get -d -v ./...
# RUN go install -v ./...

# #final stage
# FROM alpine:latest
# RUN apk --no-cache add ca-certificates
# COPY --from=builder /go/bin/app /app
# ENTRYPOINT ./app
# LABEL Name=tronicscorp Version=0.0.1
# EXPOSE 8080
