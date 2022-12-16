ARG username
ARG useruid
ARG usergid

FROM golang:1.19

ARG username
ARG useruid
ARG usergid

# Create non-root user
RUN apt-get update && \
    apt-get install -y sudo tzdata && \
    groupadd --gid ${usergid} ${username} && \
    useradd -s /bin/bash --uid ${useruid} --gid ${usergid} -m ${username} && \
    echo ${username} ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/${username} && \
    chmod 0440 /etc/sudoers.d/${username}
