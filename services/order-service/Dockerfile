# Build stage
FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /app

# Copy pom.xml and download dependencies (for better layer caching)
COPY pom.xml .
RUN mvn dependency:go-offline -B

# Copy source code and build
COPY src ./src
RUN mvn clean package -DskipTests

# Runtime stage
FROM eclipse-temurin:17-jre
WORKDIR /app

# Copy the built JAR and OpenTelemetry agent
COPY --from=build /app/target/*.jar app.jar
COPY opentelemetry-javaagent.jar opentelemetry-javaagent.jar

# Expose the application port
EXPOSE 8080

# Run with OpenTelemetry Java agent
ENTRYPOINT ["java", "-javaagent:opentelemetry-javaagent.jar", "-jar", "app.jar"] 