
git checkout master
git remote add upstream https://github.com/hashicorp/consul-template.git
git fetch upstream
git checkout master
git rebase upstream/master
git push -f origin master
