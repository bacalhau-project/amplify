import unittest
from job_runner.main import main, recursive_search, parse_args, parse_cmd_params
import tempfile
import os


class TestParseArgs(unittest.TestCase):    
    def test_parse_args_arguments_presence(self):
        with tempfile.TemporaryDirectory('', 'amplify-test-') as tmpdirname:
            with self.assertRaises(Exception) as exception_context:
                parse_args([None])
            self.assertEqual(
                str(exception_context.exception),
                "No args provided"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname])
            self.assertEqual(
                str(exception_context.exception),
                "Not enough args provided (expected 4, got 1)"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname, tmpdirname, 'foo,foo,foo'])
            self.assertEqual(
                str(exception_context.exception),
                "Not enough args provided (expected 4, got 3)"
            )

    def test_parse_args_values(self):
        with tempfile.TemporaryDirectory('', 'amplify-test-') as tmpdirname:
            with self.assertRaises(Exception) as exception_context:
                parse_args([None, 'not-a-dir', tmpdirname, 'foo', 'foo'])
            self.assertEqual(
                str(exception_context.exception),
                "Input path is not a directory"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname, 'not-a-dir', 'foo', 'foo'])
            self.assertEqual(
                str(exception_context.exception),
                "Output path is not a directory"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname, tmpdirname, '', 'ls'])
            self.assertEqual(
                str(exception_context.exception),
                "cmd_params is empty"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname, tmpdirname, 'a, ,c', 'ls'])
            self.assertEqual(
                str(exception_context.exception),
                "cmd_params contains an empty value"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname, tmpdirname, 'a,,c', 'ls'])
            self.assertEqual(
                str(exception_context.exception),
                "cmd_params contains an empty value"
            )

            with self.assertRaises(Exception) as exception_context:
                parse_args([None, tmpdirname, tmpdirname, 'foo', ''])
            self.assertEqual(
                str(exception_context.exception),
                "cmd is empty"
            )


class TestRecursiveSearch(unittest.TestCase):
    def test_recursive_search(self):
        with tempfile.TemporaryDirectory('', 'amplify-test-') as tmpdirname:
            
            with open(os.path.join(tmpdirname, 'foo.txt'), 'w') as f:
                f.write('foo')
            with open(os.path.join(tmpdirname, 'bar.txt'), 'w') as f:
                f.write('bar')
            
            # the following should be ignored
            with open(os.path.join(tmpdirname, 'empty'), 'w') as f:
                pass
            with open(os.path.join(tmpdirname, 'bar'), 'w') as f:
                f.write('bar')
            with open(os.path.join(tmpdirname, '.hidden_bar'), 'w') as f:
                f.write('hidden_bar')
            tmp_subdirname = tempfile.TemporaryDirectory('', '', tmpdirname)

            paths = recursive_search(tmpdirname)
            self.assertEqual(len(paths), 2)

            # cleanup
            tmp_subdirname.cleanup()
        
        with tempfile.TemporaryDirectory('', 'amplify-test-') as tmpdirname:
            with self.assertRaises(Exception) as exception_context:
                paths = recursive_search(tmpdirname)
            self.assertEqual(
                str(exception_context.exception),
                "No valid files found in " + tmpdirname
            )


class TestParseCmdParams(unittest.TestCase):
    def test_recursive_search(self):
        with self.assertRaises(Exception) as exception_context:
            parsed_params = parse_cmd_params("")
        self.assertEqual(
            str(exception_context.exception),
            "No valid cmd_params provided"
        )

        with self.assertRaises(Exception) as exception_context:
            parsed_params = parse_cmd_params(".")
        self.assertEqual(
            str(exception_context.exception),
            "Invalid cmd_params: ., must be file-system compliant"
        )

        with self.assertRaises(Exception) as exception_context:
            parsed_params = parse_cmd_params("42,42")
        self.assertEqual(
            str(exception_context.exception),
            "Duplicate cmd_params provided"
        )
        
        
if __name__ == '__main__':
    unittest.main()
