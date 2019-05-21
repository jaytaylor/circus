


# Command log

(note: several of these commands did not succeed)

```bash
 548  2019-01-16 19:14:22 sudo apt-get install python3-pip python3-virtualenv
 549  2019-01-16 19:16:52 virtualenv venv
 550  2019-01-16 19:17:31 python3 -m venv venv3
 551  2019-01-16 19:17:48 sudo apt-get install python3-venv
 552  2019-01-16 19:18:02 python3 -m venv venv3
 553  2019-01-16 19:18:14 . venv3/bin/activate
 554  2019-01-16 19:18:23 pip install -r requirements.txt
 555  2019-01-16 19:19:48 cat requirements.txt
 556  2019-01-16 19:20:02 pip install cytoolz
 557  2019-01-16 19:20:37 mv venv3 /tmp/
 558  2019-01-16 19:20:53 /usr/bin/python3 -m venv venv3
 559  2019-01-16 19:21:07 cp requirements.txt requirements.txt.bak
 560  2019-01-16 19:21:09 vi requirements.txt
 561  2019-01-16 19:21:17 pip install cytoolz
 562  2019-01-16 19:21:42 pip install bdist_wheel
 563  2019-01-16 19:22:26 pip install spacy
 564  2019-01-16 19:23:51 pip install wheel
 565  2019-01-16 19:23:57 pip install spacy
 566  2019-01-16 19:26:50 cat requirements.txt
 567  2019-01-16 19:27:21 cut -f1 -d'=' < requirements.txt
 568  2019-01-16 19:27:27 cut -f1 -d'=' < requirements.txt | tr $'\n' ' '
 569  2019-01-16 19:27:45 cut -f1 -d'=' < requirements.txt | grep -v '#' | tr $'\n' ' '
 570  2019-01-16 19:27:56 pip install $(cut -f1 -d'=' < requirements.txt | grep -v '#' | tr $'\n' ' ')
 571  2019-01-16 19:29:14 rm requirements.txt
 572  2019-01-16 19:29:16 pip freeze
 573  2019-01-16 19:29:23 pip freeze > requirements.txt.bak
 574  2019-01-16 19:29:33 g co requirements.txt
 575  2019-01-16 19:29:43 cp requirements.txt requirements.txt.bak
 576  2019-01-16 19:29:48 pip freeze > requirements.txt
 577  2019-01-16 19:29:58 cat requirements.txt.bak
 578  2019-01-16 19:30:08 pip install en-core-web-lg
 579  2019-01-16 19:33:59 python -m spacy download en_core_web_sm en_core_web_md en_core_web_lg
 580  2019-01-16 19:34:20 python -m spacy download en_core_web_sm
 581  2019-01-16 19:34:30 python -m spacy download en_core_web_lg
 582  2019-01-16 19:36:32 python -m spacy download en_core_web_md
 583  2019-01-16 19:51:09 history | less
 584  2019-01-16 19:51:21 exit
 585  2019-01-16 19:51:27 history | less
```

Also need:

```bash
pip install Pygments

# Pygments 'solarized-dark' theme.
#
# Needs class name modification to work with current config.toml "solarized_dark" name.
#
# git clone https://github.com/gthank/solarized-dark-pygments venv3/lib/python3.6/site-packages/pygments/styles/solarized-dark-pygments
git clone https://github.com/gthank/solarized-dark-pygments
sed -i 's/class Solarized/class Solarized_Dark/' solarized-dark-pygments/solarized.py
cp solarized-dark-pygments/solarized.py venv3/lib/python3.6/site-packages/pygments/styles/solarized_dark.py
rm -rf solarized-dark-pygments
```

