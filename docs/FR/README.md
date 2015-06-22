# cAdvisor


*Traduction faîte par https://github.com/GlobalHelp*

cAdvisor (Container Advisor) fournis une explication et des statistiques sur les performances des conteneurs. Il est un daemon en cours d'exécution qui recueille, agrégats, des processus, et exporte informations sur l'exécution des conteneurs. Spécifiquement pour chaque conteneurs il fournis un historique des usages des ressources,  ainsi qu'un histogramme des ces ressources et des statistiques sur le reseaux. Ces données sont exportées pour améliorer les performances du conteneurs.

cAdvisor a un support natif avec [Docker](https://github.com/docker/docker)et devrait supporter d'autres types de conteneurs.Les conteneurs de  cAdvisor  sont basés sur [lmctfy](https://github.com/google/lmctfy) donc les conteneurs sont surportés intrinsèquement.

![cAdvisor](https://raw.githubusercontent.com/google/cadvisor/master/logo.png "cAdvisor")

#### Lancement Rapide: Lancer cAdvisor dans in Docker Container

Pour essayer rapidement cAdvisor sur votre ordinateur avec Docker, nous avons un image de Docker qui inclus tout ce dont vous avez besoin. Vous pouvez exécuter un seul cAdvisor sur une machine pour surveiller l'ensemble des machines. Lancement:

```
sudo docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --publish=8080:8080 \
  --detach=true \
  --name=cadvisor \
  google/cadvisor:latest
```

cAdvisor est lancée maintenant (en arrière plan) à cette adresse `http://localhost:8080`. L'installation inclus les répertoires avec l'état de Docker et cAdvisor ont besoin d'être vérifier.

**Note**: Si vous utilisez centOS, Fedora, RHEL, ou encore LXC lisez vos [running instructions](docs/fr/running.md).

Nous avons détaillé [instructions](docs/fr/running.md#standalone) pour lancer cAdvisor en autonomie sans Docker. cAdvisor [running options](docs/fr/runtime_options.md) peut aussi être intéressant pour des utilisations avancés. Si vous voulez compiler votre propre cAdvisor Docker image regardez "[deployment](docs/fr/deploy.md)" page.

## Compilation Et Test

Pour plus d'instructions ,lisez [build page](docs/fr/build.md). Ceci inclus les instructions pour la compilation de le deployment d'un image *cAdvisor Docker*.

## InfluxDB and Cluster Monitoring

cAdvisor peut exporter des statistique dans [InfluxDB](http://influxdb.com). Regardez la [documentation](docs/fr/influxdb.md) pour plus d'informations et d'exemples.

cAdvisor expose aussi ses statistique dans [Prometheus](http://prometheus.io) . Regardez la [documentation](docs/prometheus.md) Pour plus d'informations.

[Heapster](https://github.com/GoogleCloudPlatform/heapster) permet un grande surveillance des conteneurs utilisant cAdvisor.

## Web UI

cAdvisor peut être consulter à cette adresse:

`http://<hostname>:<port>/`

Regardez notre [documentation](docs/fr/web.md) pour plus de details.

## Remote REST API & Clients

cAdvisor expose ses stats grâce a une REST API. Regardez l' API [documentation](docs/fr/api.md) pour plus d'informations.

Il y a un implamentation non officiel de GO-> [client](client/) directory. Regardez la [documentation](docs/fr/clients.md) pour plus d'informations.

## Feuille de route

cAdvisor vise à améliorer les caractéristiques d'utilisation et de performance ressources de conteneurs en cours d'exécution. Aujourd'hui, nous recueillons et exposons cette information aux utilisateurs. Dans notre feuille de route:
- Alerter sur la performance d'un conteneur(Exemple ->S'il ne reçoit pas assez de ressources)
- Augmente la performance du conteneur en se basant sur les conseils précédents.
- Fournis une prédiction et orchestre les différents conteneurs.

## Communautée

Les contributions ,conseils et commentaires sont les bienvenus et même encouragés! Les developeurs de cAdvisor peuvent être contacter à ces adresses: [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on [freenode.net](http://freenode.net).  Nous avons aussi [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
