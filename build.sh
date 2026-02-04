docker build -t im-demo-service .

docker run --rm \
-v "/Users/ihor/work/chat_preview/host_output:/host_output" \
im-demo-service \
cp /app/im-demo-service /host_output/



