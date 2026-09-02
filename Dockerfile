FROM eclipse-temurin:25-jre-alpine
WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser:appgroup

# Copy pre-built JAR from local build output
ARG JAR_FILE=build/libs/*[!plain].jar
COPY ${JAR_FILE} app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
