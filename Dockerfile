FROM debian:stable

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -y update &&\
     apt-get -y install python3 python3-pip sudo &&\
     pip3 install virtualenvwrapper golang &&\
     go get "github.com/cheggaaa/pb/v3" &&\
     go get "golang.org/x/crypto/pbkdf2"

RUN useradd -ms /bin/bash app &&\
    echo "export WORKON_HOME=$HOME/.virtualenvs" >> /home/app/.bashrc &&\
    mkdir -p /home/app/.virtualenvs &&\
    echo "source /usr/local/bin/virtualenvwrapper_lazy.sh" >> /home/app/.bashrc &&\
    chown -R app:app /home/app


COPY --chown=app:app app.py requirements.txt run.py run.sh static templates /home/app

RUN ["sudo", "-u", "app", "/bin/bash", "-c", "export VIRTUALENVWRAPPER_PYTHON=python3 &&\
    . /usr/local/bin/virtualenvwrapper.sh &&\
    mkvirtualenv --python=/usr/bin/python3 main -r ~/requirements.txt"]

CMD ["/usr/bin/sudo", "-u", "app", "/bin/bash", "/home/app/run.sh"]