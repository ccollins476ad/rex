# rex

```
rex [flags] <output-specifier-1> [output-specifier-n...]
```

rex is a tool that duplicates and redirects output. It is like tee, but with a few more features. It was created to solve a particular problem that tee does not: when writing to a named pipe, discard data on overflow rather than blocking.

## Examples

### Write to file

```
rex type=file,id=/tmp/myfile.txt
```

tee equivalent:

```
tee /tmp/myfile.txt >/dev/null
```

### Append to file and write to stderr

```
rex type=file,id=/tmp/myfile.txt,append type=fd,id=2
```

tee equivalent:

```
tee -a /tmp/myfile.txt >&2
```

### Write to named pipe, discard data on overflow

```
rex type=fifo,id=/tmp/myfifo,nonblocking
```

### Write twice to stdout, write to two files

```
rex type=fd,id=1 type=fd,id=1 type=file,id=/tmp/myfile1.txt,create type=file,id=/tmp/myfile2.txt,create
```

bash/tee equivalent:

```
tee /tmp/myfile1.txt /tmp/myfile2.txt | tee >(cat)
```

### Feed input to two cat processes

```
rex type=proc,id=cat type=proc,id=cat
```

bash/tee equivalent:

```
tee >(cat) >(cat) >/dev/null
```

## Flags

| flag | description |
|------|-------------|
| -b <bufsize> | Size of rex's read buffer. Default is 64KB |

## Arguments

Each argument specifies an output for rex to forward its stdin to. If the user specifies multiple outputs, rex duplicates its input for each one. An output specifier is a comma-delimited sequence of options. rex accepts the following options:

| option        | applicable types  | description |
|---------------|-------------------|-------------|
| type=t        | N/A               | Valid values of t are: fd (file descriptor), file (path), fifo (named pipe), proc (child process). |
| id=x          | all               | String that identifies the output. Path for files, fifos, and processes; integer for file descriptors. |
| create        | file, fifo        | Create the file or fifo if it does not exist. |
| append        | file              | Append to the file if it already exists. |
| perm=p        | file, fifo        | Permissions to create the file or fifo with (subject to umask). Default is 0644. |
| nonblocking   | fifo              | Discard excess data on fifo overflow. |
| args=s        | proc              | Whitespace-separated list of arguments to invoke the child process with. |
| bufsize=b     | fifo              | Configure the fifo with the given buffer size after opening it. |
