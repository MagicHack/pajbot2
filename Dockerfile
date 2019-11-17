FROM golang:buster AS build
RUN wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/microsoft.asc.gpg && wget -q https://packages.microsoft.com/config/debian/10/prod.list -O /etc/apt/sources.list.d/microsoft-prod.list
RUN curl -sL https://deb.nodesource.com/setup_12.x | bash -
RUN apt-get update && apt-get install apt-transport-https dotnet-sdk-2.2 nodejs -y
ADD . /src
RUN cd /src && ./utils/install.sh
RUN cd /src/web && npm i && npm run build
RUN cd /src/cmd/bot && go build -v -tags csharp

FROM mcr.microsoft.com/dotnet/core/runtime:2.2.7-stretch-slim
ARG COMMIT
ARG COMMIT_COUNT
ARG BRANCH
WORKDIR /app/cmd/bot
ENV LIBCOREFOLDER /usr/share/dotnet/shared/Microsoft.NETCore.App/2.2.7
ENV PB2_COMMIT=${COMMIT}
ENV PB2_COMMIT_COUNT=${COMMIT_COUNT}
ENV PB2_BRANCH=${BRANCH}
COPY --from=build /src/web/static /app/web/static
COPY --from=build /src/web/views /app/web/views
COPY --from=build /src/cmd/bot/bot /app/cmd/bot/bot
COPY --from=build /src/migrations /app/migrations/
COPY --from=build /src/cmd/bot/*.dll /app/cmd/bot/
COPY --from=build /src/cmd/bot/charmap.bin.gz /app/cmd/bot/
RUN chmod 777 /app/cmd/bot/charmap.bin.gz
CMD ["./bot"]