# TP : Maîtrise des Goroutines et de la Synchronisation en Go

Ce projet contient les solutions structurées du TP sur la concurrence et la synchronisation en Go. Toutes les solutions sont organisées sous le dossier `Ex4a`.

---

## Structure du Projet

```
Ex4a/
├── go.mod                # Module Go principal (mon_tp_goroutines)
├── README.md             # Ce rapport avec les réponses aux questions
├── exercice1/
│   └── main.go           # Sortie prématurée (sans synchronisation)
├── exercice2/
│   └── main.go           # Synchronisation avec sync.WaitGroup
├── exercice3/
│   └── main.go           # Communication par canaux (chan)
└── exercice4/
    └── main.go           # Pool de Travailleurs (Worker Pool)
```

---

## Instructions d'Exécution

Pour exécuter un exercice spécifique, positionnez-vous dans le dossier de l'exercice ou lancez la commande depuis la racine de `Ex4a` :

```bash
# Pour l'Exercice 1
go run ./exercice1

# Pour l'Exercice 2
go run ./exercice2

# Pour l'Exercice 3
go run ./exercice3

# Pour l'Exercice 4
go run ./exercice4
```

---

## Réponses aux Questions et Analyses

### Exercice 1 : Lancement Simple et Sortie Prématurée

**Question :** Que constatez-vous dans la sortie ? Est-ce que toutes les goroutines terminent leur travail avant que le programme ne s'arrête ? Expliquez pourquoi.

**Observations & Explications :**
- À l'exécution de l'Exercice 1, la seule ligne de sortie affichée est :
  ```
  Toutes les goroutines lancées.
  ```
- **Aucune** des goroutines ne commence à afficher son début de tâche ou sa fin de tâche.
- **Pourquoi ?** 
  En Go, la fin de la fonction `main()` entraîne immédiatement l'arrêt de tout le programme, y compris de toutes les goroutines encore en cours d'exécution. Les goroutines sont dites "légères" et s'exécutent en arrière-plan. Comme nous n'avons aucun mécanisme de blocage dans `main()`, le thread principal lance les 5 goroutines de manière asynchrone, affiche le message final, puis se termine instantanément. Les goroutines n'ont pas eu le temps de démarrer et de passer leur appel à `time.Sleep` avant que le programme ne soit détruit.

---

### Exercice 2 : Synchronisation avec `sync.WaitGroup`

**Question :** Le comportement du programme a-t-il changé ? Toutes les goroutines terminent-elles maintenant leur travail ?

**Observations & Explications :**
- Oui, le comportement a radicalement changé. Toutes les goroutines commencent et finissent désormais correctement leur tâche. 
- Exemple de sortie typique :
  ```
  Toutes les goroutines lancées.
  Goroutine 5: Début de la tâche...
  Goroutine 1: Début de la tâche...
  Goroutine 2: Début de la tâche...
  Goroutine 3: Début de la tâche...
  Goroutine 4: Début de la tâche...
  Goroutine 5: Tâche terminée.
  Goroutine 1: Tâche terminée.
  Goroutine 3: Tâche terminée.
  Goroutine 4: Tâche terminée.
  Goroutine 2: Tâche terminée.
  Toutes les goroutines ont terminé leur exécution.
  ```
- **Pourquoi ?** 
  L'utilisation de `sync.WaitGroup` permet au thread principal de suivre le nombre de tâches en cours :
  1. `wg.Add(1)` incrémente le compteur interne à chaque lancement.
  2. `defer wg.Done()` décrémente ce compteur dès qu'une goroutine termine (le mot-clé `defer` garantit que cette réduction a lieu même en cas de panique/erreur dans la goroutine).
  3. `wg.Wait()` bloque la fonction `main()` tant que le compteur n'est pas revenu à 0. Cela force la goroutine principale à attendre la fin de tous les traitements concurrents.

---

### Exercice 3 : Canaux et Récupération des Résultats

**Question :** Quel est l'ordre d'affichage des messages de fin de tâche et des messages de résultats ? Est-ce que l'ordre des résultats correspond à l'ordre des IDs des goroutines ? Expliquez pourquoi.

**Observations & Explications :**
- L'affichage se fait en deux étapes :
  1. D'abord, le programme affiche de manière entremêlée les démarrages et fins de tâches de chaque goroutine (ex. "Goroutine 3: Début de la tâche...", "Goroutine 1: Tâche terminée.").
  2. Ensuite, il affiche à la chaîne tous les messages de réussite (ex. "Goroutine 3 a terminé avec succès.").
- L'ordre des résultats **ne correspond pas** à l'ordre séquentiel des IDs des goroutines (1, 2, 3, 4, 5). Il correspond à l'ordre dans lequel elles ont écrit dans le canal, qui est déterminé par leur ordre de complétion (les durées de sommeil étant aléatoires).
- **Pourquoi cet ordre ?**
  - La boucle d'affichage des résultats s'exécute **après** `wg.Wait()`. Les goroutines ont donc déjà toutes fini de travailler et ont écrit dans le canal.
  - L'ordre d'écriture est non déterministe car les goroutines s'exécutent de façon concurrente et ont des temps d'attente aléatoires. Celle qui termine en premier écrit son résultat en premier.

> [!IMPORTANT]
> **Note d'Architecture : Le danger de Deadlock avec un canal non-bufferisé**
> Si le canal avait été créé non-bufferisé (`make(chan string)`), le programme se serait retrouvé en **deadlock**. Les goroutines auraient tenté d'envoyer leurs données sur le canal (`resultChan <- ...`), mais se seraient bloquées car le `main()` attendait via `wg.Wait()` avant de commencer la lecture. Le compteur du `WaitGroup` ne serait jamais descendu à zéro.
> Pour respecter la séquence demandée par le sujet (Wait puis Close puis Read), nous avons utilisé un canal bufferisé de taille 5 (`make(chan string, 5)`). Cela permet aux goroutines d'écrire leur résultat de manière non bloquante et de finir leur exécution proprement.

---

### Exercice 4 : Gestion d'un Pool de Travailleurs

**Question :** Observez l'ordre dans lequel les tâches sont traitées et les résultats sont affichés. Comment le nombre de travailleurs affecte-t-il le temps total d'exécution ?

**Observations & Explications :**
- Les tâches sont distribuées dynamiquement aux workers disponibles. Dès qu'un worker a fini sa tâche, il en prend une autre disponible sur le canal `taches`.
- L'ordre des résultats montre que les tâches ne se terminent pas dans l'ordre chronologique de leur ID, mais selon la disponibilité et la vitesse de traitement de chaque worker.
- **Impact du nombre de travailleurs sur le temps total :**
  - Si nous avons 1 seul travailleur, les tâches s'exécutent séquentiellement. Le temps total est égal à la somme des temps de chaque tâche (environ 10 * 275ms = 2.75 secondes).
  - Avec 3 travailleurs, les tâches sont traitées 3 par 3 en parallèle. Le temps total d'exécution est divisé par environ 3 (environ 2.75s / 3 = 0.9 seconde).
  - De façon générale, si $N$ est le nombre de workers et $T_{seq}$ le temps total d'une exécution séquentielle, le temps d'exécution concurrent est théoriquement proche de $\frac{T_{seq}}{N}$ (si le nombre de CPU le permet et s'il n'y a pas de goulet d'étranglement ou de contention sur les canaux).
