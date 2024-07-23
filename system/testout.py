import tempfile


class TestOut:
    def __init__(self):
        self.tmp_file = tempfile.NamedTemporaryFile(delete=False)
        self.read_pos = 0

    def fileno(self):
        return self.tmp_file.fileno()

    def write(self, text):
        self.tmp_file.write(text.encode())

    def get_contents(self):
        self.tmp_file.seek(self.read_pos, 0)
        return self.tmp_file.read().decode("utf-8")

    def close(self):
        self.tmp_file.close()

    def clear(self):
        self.read_pos = self.tmp_file.tell()
