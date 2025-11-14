#SPDX-License-Identifier: AGPL-3.0-or-later

FROM debian:trixie

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -y update &&\
     apt-get -y --no-install-recommends install python3 python3-pip sudo git libucl1 wget python3-setuptools &&\
     pip3 install --break-system-packages virtualenvwrapper &&\
     wget "https://go.dev/dl/go1.23.6.linux-amd64.tar.gz" -O "/tmp/go-linux-amd64.tar.gz" &&\
     tar -C /usr/local -xf "/tmp/go-linux-amd64.tar.gz" &&\
     rm "/tmp/go-linux-amd64.tar.gz"

RUN useradd -ms /bin/bash app &&\
    echo "export WORKON_HOME=$HOME/.virtualenvs" >> /home/app/.bashrc &&\
    mkdir -p /home/app/.virtualenvs &&\
    echo "source /usr/local/bin/virtualenvwrapper_lazy.sh" >> /home/app/.bashrc &&\
    chown -R app:app /home/app &&\
    mkdir /var/log/app && chown app:app /var/log/app

COPY --chown=app:app templates /home/app/templates
RUN sudo -u app /usr/local/go/bin/go -C /home/app/templates mod download

COPY --chown=app:app requirements.txt /home/app/

RUN ["sudo", "-u", "app", "/bin/bash", "-c", "export VIRTUALENVWRAPPER_PYTHON=python3 &&\
    . /usr/local/bin/virtualenvwrapper.sh &&\
    mkvirtualenv --python=/usr/bin/python3 main -r ~/requirements.txt"]

COPY --chown=app:app app.py run.py run.sh /home/app/
COPY static /home/app/static



EXPOSE 5000
CMD ["/usr/bin/sudo", "-u", "app", "/bin/bash", "/home/app/run.sh"]
