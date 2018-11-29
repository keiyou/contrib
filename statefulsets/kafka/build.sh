#! /bin/sh

if [ x$1 != x ];then
  docker login &&
  docker build -t steins023/kafka-test:$1 . &&
  docker push steins023/kafka-test:$1
else
  echo "Invalid Arguement: ./build.sh <tag:string>"
fi
