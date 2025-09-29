import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import subprocess
import os
import threading
import queue
import re

# --- Funciones de Lógica ---

def listar_paquetes(listbox_paquetes):
    """
    Ejecuta el comando ADB para obtener la lista de paquetes instalados
    y los muestra en el Listbox de la GUI.
    """
    try:
        resultado = subprocess.check_output(['adb', 'shell', 'pm', 'list', 'packages', '-3'], text=True, stderr=subprocess.STDOUT)
        paquetes = sorted(resultado.strip().replace("package:", "").split())

        listbox_paquetes.delete(0, tk.END)
        for paquete in paquetes:
            listbox_paquetes.insert(tk.END, paquete)

        if not paquetes:
            messagebox.showinfo("Información", "No se encontraron paquetes de terceros.")

    except FileNotFoundError:
        messagebox.showerror("Error", "Comando 'adb' no encontrado. Asegúrate de que ADB esté instalado y en el PATH del sistema.")
    except subprocess.CalledProcessError as e:
        messagebox.showerror("Error de ADB", e.output or "Dispositivo no encontrado o no autorizado.")

def iniciar_envio_archivo(paquete_seleccionado, destino_seleccionado, ventana, progress_bar, label_estado):
    """
    Prepara e inicia el hilo para enviar el archivo.
    """
    if not paquete_seleccionado:
        messagebox.showwarning("Advertencia", "Por favor, selecciona un paquete de la lista.")
        return

    ruta_archivo = filedialog.askopenfilename(title="Seleccionar archivo para enviar")
    if not ruta_archivo:
        return

    # Construye la ruta de destino
    if destino_seleccionado == "files":
        ruta_destino = f"/sdcard/Android/data/{paquete_seleccionado}/files/"
    else:
        ruta_destino = f"/sdcard/Android/obb/{paquete_seleccionado}/"

    # Prepara la barra de progreso y la cola de comunicación
    progress_bar['value'] = 0
    q = queue.Queue()

    # Inicia el hilo de trabajo
    thread = threading.Thread(target=worker_enviar_archivo, args=(ruta_archivo, ruta_destino, q))
    thread.start()

    # Inicia el chequeo periódico de la cola para actualizar la GUI
    ventana.after(100, procesar_cola_progreso, q, progress_bar, label_estado)
    label_estado.config(text=f"Enviando '{os.path.basename(ruta_archivo)}'...", fg="blue")

def worker_enviar_archivo(ruta_archivo, ruta_destino, q):
    """
    Función que se ejecuta en un hilo. Realiza el 'adb push' y
    reporta el progreso a través de la cola.
    """
    try:
        # Asegurarse de que el directorio de destino exista
        subprocess.run(['adb', 'shell', 'mkdir', '-p', ruta_destino], check=True, capture_output=True)

        comando_push = ['adb', 'push', ruta_archivo, ruta_destino]

        # Usamos Popen para capturar la salida en tiempo real
        proceso = subprocess.Popen(comando_push, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, creationflags=subprocess.CREATE_NO_WINDOW)

        # Expresión regular para encontrar el porcentaje. Ej: "[ 55%] ..."
        regex = re.compile(r"\[\s*(\d+)%\]")

        # Lee la salida línea por línea
        for linea in iter(proceso.stdout.readline, ''):
            match = regex.search(linea)
            if match:
                porcentaje = int(match.group(1))
                q.put(porcentaje) # Envía el porcentaje a la cola

        proceso.stdout.close()
        proceso.wait()

        if proceso.returncode == 0:
            q.put('DONE')
        else:
            q.put(('ERROR', 'El proceso ADB finalizó con un error.'))

    except FileNotFoundError:
        q.put(('ERROR', "Comando 'adb' no encontrado."))
    except subprocess.CalledProcessError as e:
        q.put(('ERROR', f"Error al crear directorio: {e.stderr}"))
    except Exception as e:
        q.put(('ERROR', str(e)))

def procesar_cola_progreso(q, progress_bar, label_estado):
    """
    Procesa los mensajes de la cola para actualizar la barra de progreso y el estado.
    """
    try:
        mensaje = q.get_nowait() # Lee de la cola sin bloquear

        if isinstance(mensaje, int):
            progress_bar['value'] = mensaje
        elif mensaje == 'DONE':
            progress_bar['value'] = 100
            label_estado.config(text="Estado: Archivo enviado con éxito.", fg="green")
            messagebox.showinfo("Éxito", "El archivo se ha enviado correctamente.")
            return # Termina el bucle de chequeo
        elif isinstance(mensaje, tuple) and mensaje[0] == 'ERROR':
            label_estado.config(text=f"Error: {mensaje[1]}", fg="red")
            messagebox.showerror("Error de Envío", mensaje[1])
            progress_bar['value'] = 0
            return # Termina el bucle de chequeo

    except queue.Empty: # Si la cola está vacía, no hace nada
        pass

    # Se vuelve a llamar a sí misma después de 100ms para seguir chequeando
    progress_bar.master.after(100, procesar_cola_progreso, q, progress_bar, label_estado)

# --- Creación de la Interfaz Gráfica ---

def crear_gui():
    """
    Crea y configura todos los elementos de la interfaz gráfica principal.
    """
    ventana = tk.Tk()
    ventana.title("ADB File Pusher con Progreso")
    ventana.geometry("600x550")
    ventana.resizable(False, True)

    frame_lista = tk.Frame(ventana, padx=10, pady=10)
    frame_lista.pack(fill=tk.BOTH, expand=True)

    label_paquetes = tk.Label(frame_lista, text="Paquetes Instalados (Apps de terceros):")
    label_paquetes.pack(anchor="w")

    listbox_paquetes = tk.Listbox(frame_lista, height=15)
    listbox_paquetes.pack(fill=tk.BOTH, expand=True, pady=(5,0))

    scrollbar = tk.Scrollbar(listbox_paquetes, orient="vertical", command=listbox_paquetes.yview)
    scrollbar.pack(side="right", fill="y")
    listbox_paquetes.config(yscrollcommand=scrollbar.set)

    boton_listar = tk.Button(frame_lista, text="Buscar Dispositivo y Listar Paquetes", command=lambda: listar_paquetes(listbox_paquetes))
    boton_listar.pack(fill=tk.X, pady=(10,0))

    frame_envio = tk.Frame(ventana, padx=10, pady=10)
    frame_envio.pack(fill=tk.X)

    label_destino = tk.Label(frame_envio, text="Seleccionar destino:")
    label_destino.pack(anchor="w")

    opcion_destino = tk.StringVar(value="files")
    tk.Radiobutton(frame_envio, text=".../Android/data/<paquete>/files/", variable=opcion_destino, value="files").pack(anchor="w")
    tk.Radiobutton(frame_envio, text=".../Android/obb/<paquete>/", variable=opcion_destino, value="obb").pack(anchor="w")

    # Barra de progreso
    progress_bar = ttk.Progressbar(frame_envio, orient='horizontal', length=100, mode='determinate')
    progress_bar.pack(fill=tk.X, pady=(10, 5))

    # Botón de enviar ahora llama a la función de inicio
    boton_enviar = tk.Button(frame_envio, text="Seleccionar Archivo y Enviar", command=lambda: iniciar_envio_archivo(
        listbox_paquetes.get(tk.ACTIVE) if listbox_paquetes.curselection() else None,
        opcion_destino.get(),
        ventana,
        progress_bar,
        label_estado
    ))
    boton_enviar.pack(fill=tk.X, pady=(5, 5))

    label_estado = tk.Label(ventana, text="Estado: Esperando acción.", bd=1, relief=tk.SUNKEN, anchor=tk.W, padx=5)
    label_estado.pack(side=tk.BOTTOM, fill=tk.X)

    ventana.mainloop()

if __name__ == "__main__":
    crear_gui()
