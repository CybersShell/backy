FROM debian:buster

# Install SSH server
RUN apt-get update && \
    apt-get install -y openssh-server && \
    apt-get clean

RUN sed -i 's/PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config

RUN useradd -m -s /bin/bash backy
RUN echo "backy:backy" | chpasswd
RUN echo "root:test" | chpasswd
COPY --chown=backy:backy backytest.pub /home/backy/.ssh/authorized_keys
COPY --chown=root:root backytest.pub /root/.ssh/authorized_keys

EXPOSE 22
RUN mkdir /var/run/sshd
RUN chmod 0755 /var/run/sshd

# RUN apt-get update && apt-get install -y 

# Start SSH service
CMD ["/usr/sbin/sshd", "-D"]
# ENTRYPOINT service ssh start && bash