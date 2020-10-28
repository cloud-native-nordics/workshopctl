function get_dashboard_token() {
    kubectl -n workshopctl get secret -otemplate --template {{.data.token}} $(kubectl -n workshopctl get secret | grep -o code-server-[a-z-]*) | base64 -d | xargs echo
}