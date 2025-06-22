FROM ubuntu:latest

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install SSH and set root password
RUN apt -y update && \
    apt install -y openssh-server && \
    mkdir /var/run/sshd && \
    echo "root:test" | chpasswd && \
    sed -i 's/#\?PermitRootLogin .*/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed -i 's/#\?PasswordAuthentication .*/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    echo "PermitEmptyPasswords no" >> /etc/ssh/sshd_config

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D"]