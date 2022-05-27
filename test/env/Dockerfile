# Create a gitea container that can be used for integration testing. This container:
# - Uses ephemeral data in the container (so we don't have to inject a volume)
# - Is initialized with a local sqlite database
# - Is initialized with a test user
FROM gitea/gitea:1.16.8

ENV GITEA_CUSTOM /localdata/gitea

RUN apk --no-cache add shadow \
    && usermod -d /localdata/git git \
    && apk del shadow

COPY ./entrypoint /usr/bin/entrypoint
COPY ./giteasetup /etc/s6/gitea/setup
COPY ./opensshsetup /etc/s6/openssh/setup
COPY ./sshd_config_tpl /etc/templates/sshd_config
COPY ./app.ini /localdata/gitea/conf/app.ini
RUN mkdir -p /localdata/gitea/log \
    && mkdir -p /localdata/git/.ssh \
    && mkdir -p /localdata/ssh
RUN chown -R git:git /localdata

USER git
RUN /app/gitea/gitea migrate \
    && /app/gitea/gitea admin user create \
      --username testadmin \
      --password asdf1234 \
      --email testadmin@foo.com \
      --admin
USER root
