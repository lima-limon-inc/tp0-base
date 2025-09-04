# TP0: Docker + Comunicaciones + Concurrencia

En el presente repositorio se provee un esqueleto básico de cliente/servidor, en donde todas las dependencias del mismo se encuentran encapsuladas en containers. Los alumnos deberán resolver una guía de ejercicios incrementales, teniendo en cuenta las condiciones de entrega descritas al final de este enunciado.

 El cliente (Golang) y el servidor (Python) fueron desarrollados en diferentes lenguajes simplemente para mostrar cómo dos lenguajes de programación pueden convivir en el mismo proyecto con la ayuda de containers, en este caso utilizando [Docker Compose](https://docs.docker.com/compose/).

## Instrucciones de uso
El repositorio cuenta con un **Makefile** que incluye distintos comandos en forma de targets. Los targets se ejecutan mediante la invocación de:  **make \<target\>**. Los target imprescindibles para iniciar y detener el sistema son **docker-compose-up** y **docker-compose-down**, siendo los restantes targets de utilidad para el proceso de depuración.

Los targets disponibles son:

| target  | accion  |
|---|---|
|  `docker-compose-up`  | Inicializa el ambiente de desarrollo. Construye las imágenes del cliente y el servidor, inicializa los recursos a utilizar (volúmenes, redes, etc) e inicia los propios containers. |
| `docker-compose-down`  | Ejecuta `docker-compose stop` para detener los containers asociados al compose y luego  `docker-compose down` para destruir todos los recursos asociados al proyecto que fueron inicializados. Se recomienda ejecutar este comando al finalizar cada ejecución para evitar que el disco de la máquina host se llene de versiones de desarrollo y recursos sin liberar. |
|  `docker-compose-logs` | Permite ver los logs actuales del proyecto. Acompañar con `grep` para lograr ver mensajes de una aplicación específica dentro del compose. |
| `docker-image`  | Construye las imágenes a ser utilizadas tanto en el servidor como en el cliente. Este target es utilizado por **docker-compose-up**, por lo cual se lo puede utilizar para probar nuevos cambios en las imágenes antes de arrancar el proyecto. |
| `build` | Compila la aplicación cliente para ejecución en el _host_ en lugar de en Docker. De este modo la compilación es mucho más veloz, pero requiere contar con todo el entorno de Golang y Python instalados en la máquina _host_. |

## Resoluciones

### Ejercicio 1

Para el ejercicio 1 se creo un shell script que genera el docker compose. Este contiene string "templates" que generan el docker compose usando los argumentos pasados por la CLI a la hora de llamar al script.

Decidí usar un shell script en vez de un script de python auxiliar por simplicidad.

### Ejercicio 2

Para el ejercicio 2 se añadió en el shell script la columna de "volumes".

### Ejercicio 3

En ese ejercicio escribí un shell script que usase netcat desde el container. Para hacer el script mas versátil y evitar tener los valores hardcodeados, lee los archivos de configuración para tener el puerto e IP usados.

### Ejercicio 4

En el caso del cliente, se crea un canal justo antes de empezar el loop del cliente. Este canal hace de "closure asíncrona": recibe una referencia a la instancia del cliente y llama a su método close apenas el canal reciba la señal de SIGTERM.

En el server repliqué la misma estructura, definí un closure antes de llamar a la función main del servidor. Dicho closure recibe una referencia a la instanciacion del cliente.
Luego, se bindea este closure como handler de la señal SIGTERM.

### Ejercicio 5
El protocolo de comunicación es del tipo "tamaño fijo". Los tipos primitivos se codifican de la siguiente forma:

- Un byte para el tipo de dato (actualmente: string, uint64 o uint8; siendo 0, 1 y 2 respectivamente)
- Un byte para resto de la información (en el caso del string, es variable, en los otros dos es siempre 8 y 1 respectivamente)
- N bytes para la información restante.

Esta codificación esta implementada en los archivos `client/protocol.go` y `server/common/protocol.py`.

Luego, los structs de lógica de negocio siguen un esquema similar. Particularmente, la única racionalización usada es la de las apuestas, las cuales se serializan de la siguiente manera:
- Un byte para el tipo de dato (actualmente: solo 0 para las `Bet`)
- Un uint8 para la longitud de los datos restantes
- N bytes para la información restante.
    - Esto corresponde a cada uno de los campos del struct, los cuales se serializan usando el protocolo descripto arriba.

### Ejercicio 6
En el ejercicio 6 consistió de batchear el envío de apuestas, a diferencia del ejercicio anterior donde cada apuesta se enviaba por separado.
Para esto, se agregaban n apuestas en un buffer y luego se enviaba todas juntas bajo la siguiente serializacion.
- Un byte para indicar que era un batche de apuestas (1)
- Un uint64 serializado usando la serializacion del ejercicio anterior para indicar la longitud de las apuestas serializadas.
- Las apuestas serializadas.

Cuando termina de procesar todas las apuestas se envía un byte final indicando el fin del procesamiento.

### Ejercicio 7
En este ejercicio, se tiene que realizar el sorteo después de que todas las agencias hayan publicado sus apuestas. Para esto, el servidor recibe por parámetro la cantidad de clientes que espera (esto se obtiene de los archivos de configuración o de una environment variable en su defecto).
Después de recibir y almacenar todos los batches de forma serial, se procede a procesar los ganadores.

Se leen las apuestas recibidas, buscando las apuestas ganadoras y se separan en un diccionario según la agencia que las envío. Luego, se le envía a cada agencia la lista de sus ganadores, siguiendo la siguiente serializacion:
- Un byte indicando el tipo "ganador"
- Un uint64 serializado indicando la longitud
- Las apuestas serializadas siguiendo la serializacion del 6

### Ejercicio 8
Para el ejercicio 8, se hizo uso de multithreading para el procesamiento paralelo de la información.
Esto requirió el uso de los siguientes mecanismos de sincronización:
- Un Lock para sincronizar la escritura de las bets en disco (a través de la función `store_bets`).
- Un Lock para almacenar los sockets de los clientes.
- Un Conditional variable para indicar cuando todos los clientes enviaron sus apuestas y así enviar los ganadores.

El servidor maneja la concurrencia utilizando multithreading. El hilo principal inicialmente hace la lectura de las variables de ambiente e inicializa el servidor; para luego iniciar su loop principal.

Este consiste de lo siguiente:
En el loop, el servidor se queda a la espera de la conexión de un cliente. Apenas detecta la conexión de un cliente, crea un thread nuevo y llama a la función `__handle_client_connection`. Es en este hilo donde se ejecuta la lógica de recibimiento de las apuestas por cada cliente.

Cuando el servidor detecta que llegaron todos los clientes que esperaba, comienza el procesamiento de la lotería en el thread principal; lo cual se ejecuta en la función `_handle_lottery`.
El calculo de los ganadores solo se ejecuta cuando todos los clientes enviaron el mensaje de "fin de apuesta". Esta sincronización es manejada a través de la Conditional variable `_client_finished_lock`, la cual es un integer que cada hilo cliente incrementa en 1 cuando recibe dicho mensaje.

El hecho de estar usando multithreading en Python implica que el server no le va a poder sacar el mayor provecho a los hilos, debido a las limitaciones de sincronización que el GIL impone. Sin embargo, esta limitación no debería afectar sustancialmente a la implementación debido a que cada hilo hace operaciones principalmente de I/O, las cuales [segun la documentación oficial sobre el GIL](https://wiki.python.org/moin/GlobalInterpreterLock), ocurren por fuera de las áreas afectadas por el GIL. Las únicas áreas afectadas son las áreas de procesamiento intermedio, las cuales solo implican la serializacion y desserializacion de los datos recibidos.


## Condiciones de Entrega
Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
 (hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados  [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).

# Correcciones y observaciones

## 04 de septiembre
Una de las observaciones sobre la implementación del TP fue el manejo de las conexiones. Actualmente, el servidor mantiene viva la conexion con cada uno de los clientes, mientras espera a poder procesar los ganadores.
A pesar de ser "correcto" para el alcance del TP0, este enfoque podria llegar a generar problemas si el servidor tuviese muchas conexiones simultaneamente (en el orden de las 1000 por ejemplo); lo que llevaria a que el servidor colapse.
La mejora planteada por el profesor es que el cliente se desconecte del servidor y vaya paulatinamente (utilizando un algoritmo de exponential backoff) "pingeando" al servidor para obtener su resultado.
Alternativamente, el servidor podria crear un hilo "enviador" que trate de conectarse al cliente y este le envie la informacion asi.

En terminos de correcciones, la correcion principal fue en el manejo de la señal por parte del servidor. El servidor, cuando recibe el SIGTERM, se encarga de cerrar su socket receptor, el socket de todos los cliente y marcarse como "muerto". Sin embargo, no se encarga de joinear todos los hilos, como hace en el caso de comportamiento ideal.
Entonces se anadieron las lineas:

``` python
for client in self._client_threads:
    client.join()
```

A la funcion `finalize` del servidor.

